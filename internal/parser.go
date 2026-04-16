package internal

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"
)

var (
	// Plan summary line: "Plan: 9 to add, 0 to change, 0 to destroy."
	planSummaryRe = regexp.MustCompile(`Plan:\s+(\d+)\s+to add,\s+(\d+)\s+to change,\s+(\d+)\s+to destroy`)

	// "will be created", "will be destroyed", etc.
	willCreateRe  = regexp.MustCompile(`#\s+(\S+)\s+will be created`)
	willDestroyRe = regexp.MustCompile(`#\s+(\S+)\s+will be destroyed`)
	willUpdateRe  = regexp.MustCompile(`#\s+(\S+)\s+will be updated`)
	willReplaceRe = regexp.MustCompile(`#\s+(\S+)\s+must be replaced`)
	willReadRe    = regexp.MustCompile(`#\s+(\S+)\s+will be read`)
	willImportRe  = regexp.MustCompile(`#\s+(\S+)\s+will be imported`)

	// Apply output lines
	applyCreatedRe   = regexp.MustCompile(`^(\S+):\s+Creation complete`)
	applyDestroyedRe = regexp.MustCompile(`^(\S+):\s+Destruction complete`)
	applyModifiedRe  = regexp.MustCompile(`^(\S+):\s+Modifications complete`)
	applyCreatingRe  = regexp.MustCompile(`^(\S+):\s+Creating\.\.\.`)
	applyModifyingRe = regexp.MustCompile(`^(\S+):\s+Modifying\.\.\.`)
	applyDestroyingRe = regexp.MustCompile(`^(\S+):\s+Destroying\.\.\.`)
	applyErrorRe     = regexp.MustCompile(`^Error:\s+(.+)`)
	applyResultRe    = regexp.MustCompile(`Apply complete!\s+Resources:\s+(\d+)\s+added,\s+(\d+)\s+changed,\s+(\d+)\s+destroyed`)

	// Error context: "with module.x.resource_type.name," on the line after Error:
	errorResourceRe = regexp.MustCompile(`with\s+(\S+),`)

	// Drift
	driftRe = regexp.MustCompile(`drift|Objects have changed outside of Terraform`)

	// Warning lines
	warningRe = regexp.MustCompile(`Warning:\s+(.+)`)

	// "No changes" shortcut
	noChangesRe = regexp.MustCompile(`No changes\.\s+|Your infrastructure matches the configuration`)

	// Compact resource lines: "+ module.foo.resource_type.name"
	// Match resource addresses: resource_type.name or module.x.resource_type.name
	// Must start with alphanumeric/underscore and contain at least one dot
	// Exclude quoted strings (CIDR blocks, etc.) by not matching if it starts with a quote
	compactResourceRe = regexp.MustCompile(`^\s+([+\-~])\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\[[^\]]*\])?(?:\.[a-zA-Z_][a-zA-Z0-9_]*(?:\[[^\]]*\])?)+)$`)

	// CIDR block pattern for additional validation
	cidrBlockRe = regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}/\d{1,2}$`)
)

// isValidResourceAddress checks if an address looks like a valid Terraform resource address.
// It rejects CIDR blocks, quoted strings, and other non-resource patterns.
func isValidResourceAddress(addr string) bool {
	// Reject empty addresses
	if addr == "" {
		return false
	}

	// Reject addresses that start or end with quotes
	if strings.HasPrefix(addr, `"`) || strings.HasSuffix(addr, `"`) {
		return false
	}

	// Reject addresses that start or end with single quotes
	if strings.HasPrefix(addr, `'`) || strings.HasSuffix(addr, `'`) {
		return false
	}

	// Reject CIDR blocks (e.g., "10.0.0.0/16")
	// Strip quotes first in case they're embedded
	cleanAddr := strings.Trim(addr, `"',`)
	if cidrBlockRe.MatchString(cleanAddr) {
		return false
	}

	// Reject addresses that contain CIDR notation
	if strings.Contains(addr, "/") && strings.Count(addr, ".") >= 3 {
		return false
	}

	// Valid resource addresses must start with a letter or underscore
	if len(addr) > 0 {
		first := addr[0]
		if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_') {
			return false
		}
	}

	// Valid resource addresses must contain at least one dot (resource_type.name)
	if !strings.Contains(addr, ".") {
		return false
	}

	return true
}

// Parse reads terraform plan or apply output and returns a Summary.
func Parse(input string, phase Phase, workspace string, isDestroyPlan bool) (*Summary, error) {
	s := &Summary{
		Phase:          phase,
		Workspace:      workspace,
		IsDestroyPlan:  isDestroyPlan,
	}

	// Strip ANSI codes from input for clean output
	cleanInput := stripANSI(input)
	
	// Preserve cleaned output
	s.RawOutput = cleanInput

	scanner := bufio.NewScanner(strings.NewReader(cleanInput))

	// Track in-progress apply resources to associate errors
	var lastStartedResource string
	var lastError string
	completedResources := make(map[string]bool)

	for scanner.Scan() {
		line := scanner.Text()

		// Detect drift
		if driftRe.MatchString(line) {
			s.DriftDetected = true
		}

		// Collect warnings
		if m := warningRe.FindStringSubmatch(line); len(m) > 1 {
			s.Warnings = append(s.Warnings, strings.TrimSpace(m[1]))
		}

		// Collect errors and try to associate them with a resource
		if m := applyErrorRe.FindStringSubmatch(line); len(m) > 1 {
			lastError = strings.TrimSpace(m[1])
			s.Errors = append(s.Errors, lastError)
			if s.ApplyError == "" {
				s.ApplyError = lastError
			}
		}

		// Try to extract the resource address from an error context line
		if lastError != "" {
			if m := errorResourceRe.FindStringSubmatch(line); len(m) > 1 {
				addr := stripANSI(m[1])
				// Only add if not already tracked as a failure
				if !containsAddr(s.Failures, addr) {
					s.Failures = append(s.Failures, ResourceChange{
						Address: addr,
						Action:  inferActionFromError(lastError),
						Success: false,
						Error:   lastError,
					})
				}
				lastError = ""
			}
		}

		// Parse "will be X" lines (plan output)
		if m := willCreateRe.FindStringSubmatch(line); len(m) > 1 {
			addr := stripANSI(m[1])
			if isValidResourceAddress(addr) {
				s.Creates = append(s.Creates, ResourceChange{Address: addr, Action: ActionCreate})
			}
		} else if m := willDestroyRe.FindStringSubmatch(line); len(m) > 1 {
			addr := stripANSI(m[1])
			if isValidResourceAddress(addr) {
				s.Destroys = append(s.Destroys, ResourceChange{Address: addr, Action: ActionDestroy})
			}
		} else if m := willReplaceRe.FindStringSubmatch(line); len(m) > 1 {
			addr := stripANSI(m[1])
			if isValidResourceAddress(addr) {
				s.Replaces = append(s.Replaces, ResourceChange{Address: addr, Action: ActionReplace})
			}
		} else if m := willUpdateRe.FindStringSubmatch(line); len(m) > 1 {
			addr := stripANSI(m[1])
			if isValidResourceAddress(addr) {
				s.Updates = append(s.Updates, ResourceChange{Address: addr, Action: ActionUpdate})
			}
		} else if m := willReadRe.FindStringSubmatch(line); len(m) > 1 {
			addr := stripANSI(m[1])
			if isValidResourceAddress(addr) {
				s.Reads = append(s.Reads, ResourceChange{Address: addr, Action: ActionRead})
			}
		} else if m := willImportRe.FindStringSubmatch(line); len(m) > 1 {
			addr := stripANSI(m[1])
			if isValidResourceAddress(addr) {
				s.Imports = append(s.Imports, ResourceChange{Address: addr, Action: ActionImport})
			}
		}

		// Parse compact resource lines (from plan -no-color output summary sections)
		if m := compactResourceRe.FindStringSubmatch(line); len(m) > 2 {
			addr := stripANSI(m[2])
			// Additional validation to filter out CIDR blocks and other non-resource patterns
			if isValidResourceAddress(addr) {
				switch m[1] {
				case "+":
					if !containsAddr(s.Creates, addr) {
						s.Creates = append(s.Creates, ResourceChange{Address: addr, Action: ActionCreate})
					}
				case "-":
					if !containsAddr(s.Destroys, addr) {
						s.Destroys = append(s.Destroys, ResourceChange{Address: addr, Action: ActionDestroy})
					}
				case "~":
					if !containsAddr(s.Updates, addr) {
						s.Updates = append(s.Updates, ResourceChange{Address: addr, Action: ActionUpdate})
					}
				}
			}
		}

		// Parse plan summary line
		if m := planSummaryRe.FindStringSubmatch(line); len(m) > 3 {
			s.ToAdd, _ = strconv.Atoi(m[1])
			s.ToChange, _ = strconv.Atoi(m[2])
			s.ToDestroy, _ = strconv.Atoi(m[3])
		}

		// Track apply resource lifecycle: Creating.../Modifying.../Destroying...
		if m := applyCreatingRe.FindStringSubmatch(line); len(m) > 1 {
			lastStartedResource = stripANSI(m[1])
		} else if m := applyModifyingRe.FindStringSubmatch(line); len(m) > 1 {
			lastStartedResource = stripANSI(m[1])
		} else if m := applyDestroyingRe.FindStringSubmatch(line); len(m) > 1 {
			lastStartedResource = stripANSI(m[1])
		}

		// Parse apply result
		if m := applyResultRe.FindStringSubmatch(line); len(m) > 3 {
			added, _ := strconv.Atoi(m[1])
			changed, _ := strconv.Atoi(m[2])
			destroyed, _ := strconv.Atoi(m[3])
			s.Applied = added + changed + destroyed
			s.ToAdd = added
			s.ToChange = changed
			s.ToDestroy = destroyed
			s.ApplySucceeded = true
		}

		// Parse apply resource completions
		if m := applyCreatedRe.FindStringSubmatch(line); len(m) > 1 {
			addr := stripANSI(m[1])
			s.Creates = append(s.Creates, ResourceChange{Address: addr, Action: ActionCreate, Success: true})
			completedResources[addr] = true
		}
		if m := applyDestroyedRe.FindStringSubmatch(line); len(m) > 1 {
			addr := stripANSI(m[1])
			s.Destroys = append(s.Destroys, ResourceChange{Address: addr, Action: ActionDestroy, Success: true})
			completedResources[addr] = true
		}
		if m := applyModifiedRe.FindStringSubmatch(line); len(m) > 1 {
			addr := stripANSI(m[1])
			s.Updates = append(s.Updates, ResourceChange{Address: addr, Action: ActionUpdate, Success: true})
			completedResources[addr] = true
		}

		// No changes detection - only mark as no changes if we haven't found any changes yet
		// This prevents false positives when "No changes" appears but changes were already detected
		if noChangesRe.MatchString(line) {
			// Only reset if we truly have no changes detected
			if s.ToAdd == 0 && s.ToChange == 0 && s.ToDestroy == 0 && len(s.Creates) == 0 && len(s.Updates) == 0 && len(s.Destroys) == 0 {
				// leave all zeros — renderer handles it
			}
		}
	}

	// If we had an error with no resource context line, associate it with the last started resource
	if lastError != "" && len(s.Failures) == 0 && lastStartedResource != "" {
		if !completedResources[lastStartedResource] {
			s.Failures = append(s.Failures, ResourceChange{
				Address: lastStartedResource,
				Action:  inferActionFromError(lastError),
				Success: false,
				Error:   lastError,
			})
		}
	}

	// If apply phase and we have errors but no success marker
	if phase == PhaseApply && len(s.Errors) > 0 && !s.ApplySucceeded {
		s.ApplySucceeded = false
		s.Failed = len(s.Failures)
		if s.Failed == 0 {
			s.Failed = len(s.Errors) // fallback: count errors as failures
		}
	}

	// Count imports
	s.ToImport = len(s.Imports)

	// Synchronize counters with actual resource lists to prevent drift
	// The plan summary line (ToAdd, ToChange, ToDestroy) is the authoritative source
	// but if we parsed more or fewer resources, we need to reconcile
	syncCounters(s)

	return s, nil
}

// syncCounters ensures the resource lists don't contain invalid entries.
// The plan summary line (ToAdd, ToChange, ToDestroy) is the authoritative source
// and should NOT be modified. We only filter out invalid resources from the lists.
func syncCounters(s *Summary) {
	// Filter out any invalid resources that may have slipped through parsing
	s.Creates = filterValidResources(s.Creates)
	s.Updates = filterValidResources(s.Updates)
	s.Destroys = filterValidResources(s.Destroys)
	s.Replaces = filterValidResources(s.Replaces)
	s.Imports = filterValidResources(s.Imports)
	s.Reads = filterValidResources(s.Reads)

	// Update import count based on filtered list
	s.ToImport = len(s.Imports)

	// If we have more resources in a list than the plan summary indicates,
	// trim the excess (this handles cases where invalid entries slipped through)
	if s.ToAdd > 0 && len(s.Creates) > s.ToAdd {
		s.Creates = s.Creates[:s.ToAdd]
	}
	if s.ToChange > 0 && len(s.Updates) > s.ToChange {
		s.Updates = s.Updates[:s.ToChange]
	}
	if s.ToDestroy > 0 && len(s.Destroys) > s.ToDestroy {
		s.Destroys = s.Destroys[:s.ToDestroy]
	}
}

// filterValidResources removes any resources with invalid addresses from the list.
func filterValidResources(resources []ResourceChange) []ResourceChange {
	if len(resources) == 0 {
		return resources
	}

	valid := make([]ResourceChange, 0, len(resources))
	for _, r := range resources {
		if isValidResourceAddress(r.Address) {
			valid = append(valid, r)
		}
	}
	return valid
}

// inferActionFromError guesses the action from the error message text.
func inferActionFromError(errMsg string) Action {
	lower := strings.ToLower(errMsg)
	switch {
	case strings.Contains(lower, "creating"):
		return ActionCreate
	case strings.Contains(lower, "destroying") || strings.Contains(lower, "deleting"):
		return ActionDestroy
	case strings.Contains(lower, "updating") || strings.Contains(lower, "modifying"):
		return ActionUpdate
	default:
		return ActionCreate
	}
}

func containsAddr(changes []ResourceChange, addr string) bool {
	for _, c := range changes {
		if c.Address == addr {
			return true
		}
	}
	return false
}

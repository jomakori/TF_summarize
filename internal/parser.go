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
	// Require a dot in the address to avoid matching action keywords like "+ create"
	compactResourceRe = regexp.MustCompile(`^\s+([+\-~])\s+(\S+\.\S+)$`)
)

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
			s.Creates = append(s.Creates, ResourceChange{Address: addr, Action: ActionCreate})
		} else if m := willDestroyRe.FindStringSubmatch(line); len(m) > 1 {
			addr := stripANSI(m[1])
			s.Destroys = append(s.Destroys, ResourceChange{Address: addr, Action: ActionDestroy})
		} else if m := willReplaceRe.FindStringSubmatch(line); len(m) > 1 {
			addr := stripANSI(m[1])
			s.Replaces = append(s.Replaces, ResourceChange{Address: addr, Action: ActionReplace})
		} else if m := willUpdateRe.FindStringSubmatch(line); len(m) > 1 {
			addr := stripANSI(m[1])
			s.Updates = append(s.Updates, ResourceChange{Address: addr, Action: ActionUpdate})
		} else if m := willReadRe.FindStringSubmatch(line); len(m) > 1 {
			addr := stripANSI(m[1])
			s.Reads = append(s.Reads, ResourceChange{Address: addr, Action: ActionRead})
		} else if m := willImportRe.FindStringSubmatch(line); len(m) > 1 {
			addr := stripANSI(m[1])
			s.Imports = append(s.Imports, ResourceChange{Address: addr, Action: ActionImport})
		}

		// Parse compact resource lines (from plan -no-color output summary sections)
		if m := compactResourceRe.FindStringSubmatch(line); len(m) > 2 {
			addr := stripANSI(m[2])
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

	return s, nil
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

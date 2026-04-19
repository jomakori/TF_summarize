package parser

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"

	"github.com/jomakori/TF_summarize/internal"
)

// Regex patterns for parsing terraform output.
var (
	planSummaryRe     = regexp.MustCompile(`Plan:\s+(\d+)\s+to add,\s+(\d+)\s+to change,\s+(\d+)\s+to destroy`)
	willCreateRe      = regexp.MustCompile(`#\s+(\S+)\s+will be created`)
	willDestroyRe     = regexp.MustCompile(`#\s+(\S+)\s+will be destroyed`)
	willUpdateRe      = regexp.MustCompile(`#\s+(\S+)\s+will be updated`)
	willReplaceRe     = regexp.MustCompile(`#\s+(\S+)\s+must be replaced`)
	willReadRe        = regexp.MustCompile(`#\s+(\S+)\s+will be read`)
	willImportRe      = regexp.MustCompile(`#\s+(\S+)\s+will be imported`)
	applyCreatedRe    = regexp.MustCompile(`^(\S+):\s+Creation complete`)
	applyDestroyedRe  = regexp.MustCompile(`^(\S+):\s+Destruction complete`)
	applyModifiedRe   = regexp.MustCompile(`^(\S+):\s+Modifications complete`)
	applyCreatingRe   = regexp.MustCompile(`^(\S+):\s+Creating\.\.\.`)
	applyModifyingRe  = regexp.MustCompile(`^(\S+):\s+Modifying\.\.\.`)
	applyDestroyingRe = regexp.MustCompile(`^(\S+):\s+Destroying\.\.\.`)
	// Match both standard "Error: msg" and box format "│ Error: msg"
	applyErrorRe      = regexp.MustCompile(`(?:^│\s*)?Error:\s+(.+)`)
	applyResultRe     = regexp.MustCompile(`Apply complete!\s+Resources:\s+(\d+)\s+added,\s+(\d+)\s+changed,\s+(\d+)\s+destroyed`)
	errorResourceRe   = regexp.MustCompile(`with\s+(\S+),`)
	driftRe           = regexp.MustCompile(`drift|Objects have changed outside of Terraform`)
	warningRe         = regexp.MustCompile(`Warning:\s+(.+)`)
	noChangesRe       = regexp.MustCompile(`No changes\.\s+|Your infrastructure matches the configuration`)
	compactResourceRe = regexp.MustCompile(`^\s+([+\-~])\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\[[^\]]*\])?(?:\.[a-zA-Z_][a-zA-Z0-9_]*(?:\[[^\]]*\])?)+)$`)
)

// Parse reads terraform plan or apply output and returns a Summary.
func Parse(input string, phase internal.Phase, workspace string, isDestroyPlan bool) (*internal.Summary, error) {
	s := &internal.Summary{
		Phase:         phase,
		Workspace:     workspace,
		IsDestroyPlan: isDestroyPlan,
	}

	cleanInput := internal.StripANSI(input)
	s.RawOutput = cleanInput
	scanner := bufio.NewScanner(strings.NewReader(cleanInput))

	var lastStartedResource string
	var lastError string
	completedResources := make(map[string]bool)

	for scanner.Scan() {
		line := scanner.Text()

		if driftRe.MatchString(line) {
			s.DriftDetected = true
		}

		if m := warningRe.FindStringSubmatch(line); len(m) > 1 {
			s.Warnings = append(s.Warnings, strings.TrimSpace(m[1]))
		}

		if m := applyErrorRe.FindStringSubmatch(line); len(m) > 1 {
			lastError = strings.TrimSpace(m[1])
			s.Errors = append(s.Errors, lastError)
			if s.ApplyError == "" {
				s.ApplyError = lastError
			}
		}

		if lastError != "" {
			if m := errorResourceRe.FindStringSubmatch(line); len(m) > 1 {
				addr := internal.StripANSI(m[1])
				if !internal.ContainsResourceAddr(s.Failures, addr) {
					s.Failures = append(s.Failures, internal.ResourceChange{
						Address: addr,
						Action:  inferActionFromError(lastError),
						Success: false,
						Error:   lastError,
					})
				}
				lastError = ""
			}
		}

		if m := willCreateRe.FindStringSubmatch(line); len(m) > 1 {
			addr := internal.StripANSI(m[1])
			if internal.IsValidResourceAddress(addr) {
				s.Creates = append(s.Creates, internal.ResourceChange{Address: addr, Action: internal.ActionCreate})
			}
		} else if m := willDestroyRe.FindStringSubmatch(line); len(m) > 1 {
			addr := internal.StripANSI(m[1])
			if internal.IsValidResourceAddress(addr) {
				s.Destroys = append(s.Destroys, internal.ResourceChange{Address: addr, Action: internal.ActionDestroy})
			}
		} else if m := willReplaceRe.FindStringSubmatch(line); len(m) > 1 {
			addr := internal.StripANSI(m[1])
			if internal.IsValidResourceAddress(addr) {
				s.Replaces = append(s.Replaces, internal.ResourceChange{Address: addr, Action: internal.ActionReplace})
			}
		} else if m := willUpdateRe.FindStringSubmatch(line); len(m) > 1 {
			addr := internal.StripANSI(m[1])
			if internal.IsValidResourceAddress(addr) {
				s.Updates = append(s.Updates, internal.ResourceChange{Address: addr, Action: internal.ActionUpdate})
			}
		} else if m := willReadRe.FindStringSubmatch(line); len(m) > 1 {
			addr := internal.StripANSI(m[1])
			if internal.IsValidResourceAddress(addr) {
				s.Reads = append(s.Reads, internal.ResourceChange{Address: addr, Action: internal.ActionRead})
			}
		} else if m := willImportRe.FindStringSubmatch(line); len(m) > 1 {
			addr := internal.StripANSI(m[1])
			if internal.IsValidResourceAddress(addr) {
				s.Imports = append(s.Imports, internal.ResourceChange{Address: addr, Action: internal.ActionImport})
			}
		}

		if m := compactResourceRe.FindStringSubmatch(line); len(m) > 2 {
			addr := internal.StripANSI(m[2])
			if internal.IsValidResourceAddress(addr) {
				switch m[1] {
				case "+":
					if !internal.ContainsResourceAddr(s.Creates, addr) {
						s.Creates = append(s.Creates, internal.ResourceChange{Address: addr, Action: internal.ActionCreate})
					}
				case "-":
					if !internal.ContainsResourceAddr(s.Destroys, addr) {
						s.Destroys = append(s.Destroys, internal.ResourceChange{Address: addr, Action: internal.ActionDestroy})
					}
				case "~":
					if !internal.ContainsResourceAddr(s.Updates, addr) {
						s.Updates = append(s.Updates, internal.ResourceChange{Address: addr, Action: internal.ActionUpdate})
					}
				}
			}
		}

		if m := planSummaryRe.FindStringSubmatch(line); len(m) > 3 {
			s.ToAdd, _ = strconv.Atoi(m[1])
			s.ToChange, _ = strconv.Atoi(m[2])
			s.ToDestroy, _ = strconv.Atoi(m[3])
		}

		if m := applyCreatingRe.FindStringSubmatch(line); len(m) > 1 {
			lastStartedResource = internal.StripANSI(m[1])
		} else if m := applyModifyingRe.FindStringSubmatch(line); len(m) > 1 {
			lastStartedResource = internal.StripANSI(m[1])
		} else if m := applyDestroyingRe.FindStringSubmatch(line); len(m) > 1 {
			lastStartedResource = internal.StripANSI(m[1])
		}

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

		if m := applyCreatedRe.FindStringSubmatch(line); len(m) > 1 {
			addr := internal.StripANSI(m[1])
			s.Creates = append(s.Creates, internal.ResourceChange{Address: addr, Action: internal.ActionCreate, Success: true})
			completedResources[addr] = true
		}
		if m := applyDestroyedRe.FindStringSubmatch(line); len(m) > 1 {
			addr := internal.StripANSI(m[1])
			s.Destroys = append(s.Destroys, internal.ResourceChange{Address: addr, Action: internal.ActionDestroy, Success: true})
			completedResources[addr] = true
		}
		if m := applyModifiedRe.FindStringSubmatch(line); len(m) > 1 {
			addr := internal.StripANSI(m[1])
			s.Updates = append(s.Updates, internal.ResourceChange{Address: addr, Action: internal.ActionUpdate, Success: true})
			completedResources[addr] = true
		}

		if noChangesRe.MatchString(line) {
			if s.ToAdd == 0 && s.ToChange == 0 && s.ToDestroy == 0 && len(s.Creates) == 0 && len(s.Updates) == 0 && len(s.Destroys) == 0 {
				// No changes - renderer handles it
			}
		}
	}

	if lastError != "" && len(s.Failures) == 0 && lastStartedResource != "" {
		if !completedResources[lastStartedResource] {
			s.Failures = append(s.Failures, internal.ResourceChange{
				Address: lastStartedResource,
				Action:  inferActionFromError(lastError),
				Success: false,
				Error:   lastError,
			})
		}
	}

	if phase == internal.PhaseApply && len(s.Errors) > 0 && !s.ApplySucceeded {
		s.ApplySucceeded = false
		s.Failed = len(s.Failures)
		if s.Failed == 0 {
			s.Failed = len(s.Errors)
		}
	}

	s.ToImport = len(s.Imports)
	syncCounters(s)

	return s, nil
}

// syncCounters filters invalid resources and trims excess entries.
func syncCounters(s *internal.Summary) {
	s.Creates = internal.FilterValidResources(s.Creates)
	s.Updates = internal.FilterValidResources(s.Updates)
	s.Destroys = internal.FilterValidResources(s.Destroys)
	s.Replaces = internal.FilterValidResources(s.Replaces)
	s.Imports = internal.FilterValidResources(s.Imports)
	s.Reads = internal.FilterValidResources(s.Reads)
	s.ToImport = len(s.Imports)

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

// inferActionFromError guesses the action from error message text.
func inferActionFromError(errMsg string) internal.Action {
	lower := strings.ToLower(errMsg)
	switch {
	case strings.Contains(lower, "creating"):
		return internal.ActionCreate
	case strings.Contains(lower, "destroying") || strings.Contains(lower, "deleting"):
		return internal.ActionDestroy
	case strings.Contains(lower, "updating") || strings.Contains(lower, "modifying"):
		return internal.ActionUpdate
	default:
		return internal.ActionCreate
	}
}

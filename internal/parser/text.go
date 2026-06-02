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
	// Match multiple error formats:
	// - Standard: "Error: msg"
	// - Box format: "│ Error: msg"
	// - GitHub Actions annotation: "::error::msg"
	applyErrorRe      = regexp.MustCompile(`(?:^│\s*)?Error:\s+(.+)|^::error::(.+)`)
	applyResultRe     = regexp.MustCompile(`Apply complete!\s+Resources:\s+(\d+)\s+added,\s+(\d+)\s+changed,\s+(\d+)\s+destroyed`)
	errorResourceRe   = regexp.MustCompile(`with\s+(\S+),`)
	driftRe  = regexp.MustCompile(`drift|Objects have changed outside of Terraform`)
	msgRe    = regexp.MustCompile(`(Warning|Caution|Note|Error):\s+(.+)`)
	msgLineRe = regexp.MustCompile(`^\s+[│╷ ]\s*(.+)$|^\s{2,}(.+)$`)
	noChangesRe       = regexp.MustCompile(`No changes\.\s+|Your infrastructure matches the configuration`)
	compactResourceRe = regexp.MustCompile(`^\s+([+\-~])\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\[[^\]]*\])?(?:\.[a-zA-Z_][a-zA-Z0-9_]*(?:\[[^\]]*\])?)+)$`)
	// Output changes: "+ output_name = value" or "- output_name = value" or "~ output_name = value"
	outputChangeRe    = regexp.MustCompile(`^\s*([+\-~])\s+(\w+)\s+=\s+(.+)$`)
	outputsSectionRe  = regexp.MustCompile(`Changes to Outputs:`)
)

// msgCapture holds state for multi-line message context capture
type msgCapture struct {
	lastMsg     string
	msgType     string // "Warning", "Caution", "Note", "Error"
}

// isContextLine checks if line is a continuation of message context
func isContextLine(line string) bool {
	return strings.HasPrefix(line, " ") || strings.HasPrefix(line, "│") || strings.HasPrefix(line, "╷")
}

// flushMessage stores the captured message to the summary
func flushMessage(c *msgCapture, s *internal.Summary, lastStartedResource string, completedResources map[string]bool) {
	if c.lastMsg == "" {
		return
	}

	switch c.msgType {
	case "Warning":
		s.Warnings = append(s.Warnings, c.lastMsg)
	case "Caution", "Note":
		s.Warnings = append(s.Warnings, c.lastMsg)
	case "Error":
		s.Errors = append(s.Errors, c.lastMsg)
		if s.ApplyError == "" {
			s.ApplyError = c.lastMsg
		}
	}
	c.lastMsg = ""
	c.msgType = ""
}

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
	var msgCapture msgCapture
	completedResources := make(map[string]bool)
	inOutputsSection := false

	for scanner.Scan() {
		line := scanner.Text()

		// Handle multi-line message context capture
		if msgCapture.lastMsg != "" {
			// Check for resource address first (even on indented lines)
			if msgCapture.msgType == "Error" {
				if m := errorResourceRe.FindStringSubmatch(line); len(m) > 1 {
					addr := internal.StripANSI(m[1])
					if !internal.ContainsResourceAddr(s.Failures, addr) {
						s.Failures = append(s.Failures, internal.ResourceChange{
							Address: addr,
							Action:  inferActionFromError(msgCapture.lastMsg),
							Success: false,
							Error:   msgCapture.lastMsg,
						})
					}
					s.Errors = append(s.Errors, msgCapture.lastMsg)
					if s.ApplyError == "" {
						s.ApplyError = msgCapture.lastMsg
					}
					msgCapture.lastMsg = ""
					msgCapture.msgType = ""
					continue
				}
			}
			// Otherwise, treat as context continuation (including empty lines)
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || isContextLine(line) {
				if trimmed != "" {
					msgCapture.lastMsg += "\n" + trimmed
				}
				continue
			}
			flushMessage(&msgCapture, s, lastStartedResource, completedResources)
		}

		// Check for "Changes to Outputs:" section
		if outputsSectionRe.MatchString(line) {
			inOutputsSection = true
			continue
		}

		// Parse output changes when in outputs section
		if inOutputsSection {
			// Empty line or new section ends outputs section
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "Plan:") || strings.HasPrefix(trimmed, "─") {
				inOutputsSection = false
			} else if m := outputChangeRe.FindStringSubmatch(line); len(m) > 3 {
				action := internal.ActionCreate
				switch m[1] {
				case "+":
					action = internal.ActionCreate
				case "-":
					action = internal.ActionDestroy
				case "~":
					action = internal.ActionUpdate
				}
				s.Outputs = append(s.Outputs, internal.OutputChange{
					Name:   m[2],
					Action: action,
					Value:  strings.TrimSpace(m[3]),
				})
				continue
			}
		}

		if driftRe.MatchString(line) {
			s.DriftDetected = true
		}

		// Centralized message parsing (Warning, Caution, Note, Error)
		if m := msgRe.FindStringSubmatch(line); len(m) > 2 {
			msgCapture.msgType = m[1]
			msgCapture.lastMsg = strings.TrimSpace(m[2])
			continue
		}

		// Legacy error pattern for additional coverage
		if msgCapture.msgType != "Error" {
			if m := applyErrorRe.FindStringSubmatch(line); len(m) > 1 {
				var errMsg string
				if m[1] != "" {
					errMsg = strings.TrimSpace(m[1])
				} else if len(m) > 2 && m[2] != "" {
					errMsg = strings.TrimSpace(m[2])
				}
				if errMsg != "" {
					msgCapture.msgType = "Error"
					msgCapture.lastMsg = errMsg
					continue
				}
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

	if msgCapture.lastMsg != "" {
		if msgCapture.msgType == "Error" && len(s.Failures) == 0 && lastStartedResource != "" {
			if !completedResources[lastStartedResource] {
				s.Failures = append(s.Failures, internal.ResourceChange{
					Address: lastStartedResource,
					Action:  inferActionFromError(msgCapture.lastMsg),
					Success: false,
					Error:   msgCapture.lastMsg,
				})
			}
		}
		flushMessage(&msgCapture, s, lastStartedResource, completedResources)
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

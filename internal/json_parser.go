package internal

import (
	"encoding/json"
	"fmt"
	"time"

	tfjson "github.com/hashicorp/terraform-json"
)

// ParsePlanJSON parses a terraform plan JSON file and returns a Summary.
// This provides structured, accurate parsing compared to regex-based text parsing.
func ParsePlanJSON(jsonData []byte, workspace string, isDestroyPlan bool) (*Summary, error) {
	var plan tfjson.Plan
	if err := json.Unmarshal(jsonData, &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan JSON: %w", err)
	}

	s := &Summary{
		Phase:          PhasePlan,
		Workspace:      workspace,
		IsDestroyPlan:  isDestroyPlan,
		ParsedFromJSON: true,
	}

	// Process resource changes
	if plan.ResourceChanges != nil {
		processResourceChangesJSON(plan.ResourceChanges, s)
	}

	// Process output changes (OutputChanges is map[string]*tfjson.Change)
	if plan.OutputChanges != nil {
		// Output changes are tracked but don't affect resource counts
		// This is handled in the renderer
	}

	return s, nil
}

// ParseApplyJSON parses terraform apply output in JSON format.
// Note: terraform apply doesn't output JSON by default, but this supports
// structured apply data if available from other sources.
func ParseApplyJSON(jsonData []byte, workspace string) (*Summary, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal apply JSON: %w", err)
	}

	s := &Summary{
		Phase:          PhaseApply,
		Workspace:      workspace,
		ParsedFromJSON: true,
	}

	// Extract counts if available
	if resources, ok := data["resources"].(map[string]interface{}); ok {
		s.ToAdd = countByAction(resources, "create")
		s.ToChange = countByAction(resources, "update")
		s.ToDestroy = countByAction(resources, "destroy")
	}

	return s, nil
}

// processResourceChangesJSON processes resource changes from a terraform plan JSON.
func processResourceChangesJSON(changes []*tfjson.ResourceChange, s *Summary) {
	now := time.Now()

	for _, rc := range changes {
		if rc == nil || rc.Change == nil {
			continue
		}

		// Determine action based on change actions
		action := determineActionFromChange(rc.Change)
		if action == "" {
			continue
		}

		addr := rc.Address
		change := ResourceChange{
			Address:   addr,
			Action:    action,
			Success:   true,
			Timestamp: now,
			Details:   make(map[string]interface{}),
		}

		// Store change details
		if rc.Change.Before != nil {
			change.Details["before"] = rc.Change.Before
		}
		if rc.Change.After != nil {
			change.Details["after"] = rc.Change.After
		}

		// Categorize by action
		switch action {
		case ActionCreate:
			s.Creates = append(s.Creates, change)
			s.ToAdd++
		case ActionUpdate:
			s.Updates = append(s.Updates, change)
			s.ToChange++
		case ActionDestroy:
			s.Destroys = append(s.Destroys, change)
			s.ToDestroy++
		case ActionReplace:
			s.Replaces = append(s.Replaces, change)
			s.ToDestroy++ // Replace counts as a destroy
			s.ToAdd++     // and a create
		case ActionRead:
			s.Reads = append(s.Reads, change)
		case ActionImport:
			s.Imports = append(s.Imports, change)
			s.ToImport++
		}
	}
}

// determineActionFromChange determines the action from a terraform change.
func determineActionFromChange(change *tfjson.Change) Action {
	if change == nil {
		return ""
	}

	// Check the Actions field which contains the operations
	if len(change.Actions) == 0 {
		return ""
	}

	// Single action
	if len(change.Actions) == 1 {
		switch change.Actions[0] {
		case tfjson.ActionCreate:
			return ActionCreate
		case tfjson.ActionUpdate:
			return ActionUpdate
		case tfjson.ActionDelete:
			return ActionDestroy
		case tfjson.ActionRead:
			return ActionRead
		case tfjson.ActionNoop:
			return ""
		}
	}

	// Multiple actions indicate a replace (delete + create)
	if len(change.Actions) == 2 {
		hasDelete := false
		hasCreate := false
		for _, action := range change.Actions {
			if action == tfjson.ActionDelete {
				hasDelete = true
			}
			if action == tfjson.ActionCreate {
				hasCreate = true
			}
		}
		if hasDelete && hasCreate {
			return ActionReplace
		}
	}

	return ""
}

// countByAction counts resources by action type in a resources map.
func countByAction(resources map[string]interface{}, actionType string) int {
	count := 0
	for _, v := range resources {
		if m, ok := v.(map[string]interface{}); ok {
			if action, ok := m["action"].(string); ok && action == actionType {
				count++
			}
		}
	}
	return count
}

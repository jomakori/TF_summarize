package parser

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jomakori/TF_summarize/internal"
	tfjson "github.com/hashicorp/terraform-json"
)

// ParsePlanJSON parses a terraform plan JSON file and returns a Summary.
func ParsePlanJSON(jsonData []byte, workspace string, isDestroyPlan bool) (*internal.Summary, error) {
	var plan tfjson.Plan
	if err := json.Unmarshal(jsonData, &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan JSON: %w", err)
	}

	s := &internal.Summary{
		Phase:          internal.PhasePlan,
		Workspace:      workspace,
		IsDestroyPlan:  isDestroyPlan,
		ParsedFromJSON: true,
	}

	if plan.ResourceChanges != nil {
		processResourceChangesJSON(plan.ResourceChanges, s)
	}

	return s, nil
}

// ParseApplyJSON parses terraform apply output in JSON format.
func ParseApplyJSON(jsonData []byte, workspace string) (*internal.Summary, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal apply JSON: %w", err)
	}

	s := &internal.Summary{
		Phase:          internal.PhaseApply,
		Workspace:      workspace,
		ParsedFromJSON: true,
	}

	if resources, ok := data["resources"].(map[string]interface{}); ok {
		s.ToAdd = countByAction(resources, "create")
		s.ToChange = countByAction(resources, "update")
		s.ToDestroy = countByAction(resources, "destroy")
	}

	return s, nil
}

// processResourceChangesJSON processes resource changes from a terraform plan JSON.
func processResourceChangesJSON(changes []*tfjson.ResourceChange, s *internal.Summary) {
	now := time.Now()

	for _, rc := range changes {
		if rc == nil || rc.Change == nil {
			continue
		}

		action := determineActionFromChange(rc.Change)
		if action == "" {
			continue
		}

		addr := rc.Address
		change := internal.ResourceChange{
			Address:   addr,
			Action:    action,
			Success:   true,
			Timestamp: now,
			Details:   make(map[string]interface{}),
		}

		if rc.Change.Before != nil {
			change.Details["before"] = rc.Change.Before
		}
		if rc.Change.After != nil {
			change.Details["after"] = rc.Change.After
		}

		switch action {
		case internal.ActionCreate:
			s.Creates = append(s.Creates, change)
			s.ToAdd++
		case internal.ActionUpdate:
			s.Updates = append(s.Updates, change)
			s.ToChange++
		case internal.ActionDestroy:
			s.Destroys = append(s.Destroys, change)
			s.ToDestroy++
		case internal.ActionReplace:
			s.Replaces = append(s.Replaces, change)
			s.ToDestroy++
			s.ToAdd++
		case internal.ActionRead:
			s.Reads = append(s.Reads, change)
		case internal.ActionImport:
			s.Imports = append(s.Imports, change)
			s.ToImport++
		}
	}
}

// determineActionFromChange determines the action from a terraform change.
func determineActionFromChange(change *tfjson.Change) internal.Action {
	if change == nil {
		return ""
	}

	if len(change.Actions) == 0 {
		return ""
	}

	if len(change.Actions) == 1 {
		switch change.Actions[0] {
		case tfjson.ActionCreate:
			return internal.ActionCreate
		case tfjson.ActionUpdate:
			return internal.ActionUpdate
		case tfjson.ActionDelete:
			return internal.ActionDestroy
		case tfjson.ActionRead:
			return internal.ActionRead
		case tfjson.ActionNoop:
			return ""
		}
	}

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
			return internal.ActionReplace
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

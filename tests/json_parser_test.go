package tests

import (
	"testing"

	"github.com/jomakori/TF_summarize/internal"
	"github.com/jomakori/TF_summarize/internal/parser"
)

// TestParseSimplePlanJSON tests parsing a simple terraform plan JSON.
func TestParseSimplePlanJSON(t *testing.T) {
	// Minimal valid terraform plan JSON
	jsonData := []byte(`{
		"format_version": "1.2",
		"terraform_version": "1.5.0",
		"planned_values": {},
		"resource_changes": [
			{
				"address": "aws_instance.example",
				"mode": "managed",
				"type": "aws_instance",
				"name": "example",
				"provider_name": "registry.terraform.io/hashicorp/aws",
				"change": {
					"actions": ["create"],
					"before": null,
					"after": {
						"ami": "ami-12345678",
						"instance_type": "t2.micro"
					},
					"after_unknown": {},
					"before_sensitive": false,
					"after_sensitive": {}
				}
			},
			{
				"address": "aws_instance.old",
				"mode": "managed",
				"type": "aws_instance",
				"name": "old",
				"provider_name": "registry.terraform.io/hashicorp/aws",
				"change": {
					"actions": ["delete"],
					"before": {
						"ami": "ami-87654321",
						"instance_type": "t2.small"
					},
					"after": null,
					"after_unknown": {},
					"before_sensitive": false,
					"after_sensitive": {}
				}
			}
		],
		"output_changes": {},
		"resource_drift": null,
		"configuration": {}
	}`)

	summary, err := parser.ParsePlanJSON(jsonData, "test", false)
	if err != nil {
		t.Fatalf("ParsePlanJSON failed: %v", err)
	}

	// Verify summary
	if summary.Workspace != "test" {
		t.Errorf("Expected workspace 'test', got '%s'", summary.Workspace)
	}

	if summary.ToAdd != 1 {
		t.Errorf("Expected 1 resource to add, got %d", summary.ToAdd)
	}

	if summary.ToDestroy != 1 {
		t.Errorf("Expected 1 resource to destroy, got %d", summary.ToDestroy)
	}

	if len(summary.Creates) != 1 {
		t.Errorf("Expected 1 create, got %d", len(summary.Creates))
	}

	if len(summary.Destroys) != 1 {
		t.Errorf("Expected 1 destroy, got %d", len(summary.Destroys))
	}

	if !summary.ParsedFromJSON {
		t.Error("Expected ParsedFromJSON to be true")
	}

	// Verify resource addresses
	if summary.Creates[0].Address != "aws_instance.example" {
		t.Errorf("Expected create address 'aws_instance.example', got '%s'", summary.Creates[0].Address)
	}

	if summary.Destroys[0].Address != "aws_instance.old" {
		t.Errorf("Expected destroy address 'aws_instance.old', got '%s'", summary.Destroys[0].Address)
	}
}

// TestParseReplaceJSON tests parsing a replace operation (delete + create).
func TestParseReplaceJSON(t *testing.T) {
	jsonData := []byte(`{
		"format_version": "1.2",
		"terraform_version": "1.5.0",
		"planned_values": {},
		"resource_changes": [
			{
				"address": "aws_instance.web",
				"mode": "managed",
				"type": "aws_instance",
				"name": "web",
				"provider_name": "registry.terraform.io/hashicorp/aws",
				"change": {
					"actions": ["delete", "create"],
					"before": {
						"ami": "ami-old",
						"instance_type": "t2.micro"
					},
					"after": {
						"ami": "ami-new",
						"instance_type": "t2.small"
					},
					"after_unknown": {},
					"before_sensitive": false,
					"after_sensitive": {}
				}
			}
		],
		"output_changes": {},
		"resource_drift": null,
		"configuration": {}
	}`)

	summary, err := parser.ParsePlanJSON(jsonData, "test", false)
	if err != nil {
		t.Fatalf("ParsePlanJSON failed: %v", err)
	}

	if len(summary.Replaces) != 1 {
		t.Errorf("Expected 1 replace, got %d", len(summary.Replaces))
	}

	if summary.Replaces[0].Address != "aws_instance.web" {
		t.Errorf("Expected replace address 'aws_instance.web', got '%s'", summary.Replaces[0].Address)
	}

	if summary.Replaces[0].Action != internal.ActionReplace {
		t.Errorf("Expected action 'replace', got '%s'", summary.Replaces[0].Action)
	}
}

// TestParseUpdateJSON tests parsing an update operation.
func TestParseUpdateJSON(t *testing.T) {
	jsonData := []byte(`{
		"format_version": "1.2",
		"terraform_version": "1.5.0",
		"planned_values": {},
		"resource_changes": [
			{
				"address": "aws_instance.web",
				"mode": "managed",
				"type": "aws_instance",
				"name": "web",
				"provider_name": "registry.terraform.io/hashicorp/aws",
				"change": {
					"actions": ["update"],
					"before": {
						"instance_type": "t2.micro"
					},
					"after": {
						"instance_type": "t2.small"
					},
					"after_unknown": {},
					"before_sensitive": false,
					"after_sensitive": {}
				}
			}
		],
		"output_changes": {},
		"resource_drift": null,
		"configuration": {}
	}`)

	summary, err := parser.ParsePlanJSON(jsonData, "test", false)
	if err != nil {
		t.Fatalf("ParsePlanJSON failed: %v", err)
	}

	if summary.ToChange != 1 {
		t.Errorf("Expected 1 resource to change, got %d", summary.ToChange)
	}

	if len(summary.Updates) != 1 {
		t.Errorf("Expected 1 update, got %d", len(summary.Updates))
	}

	if summary.Updates[0].Action != internal.ActionUpdate {
		t.Errorf("Expected action 'update', got '%s'", summary.Updates[0].Action)
	}
}

// TestParseNoChangesJSON tests parsing a plan with no changes.
func TestParseNoChangesJSON(t *testing.T) {
	jsonData := []byte(`{
		"format_version": "1.2",
		"terraform_version": "1.5.0",
		"planned_values": {},
		"resource_changes": [],
		"output_changes": {},
		"resource_drift": null,
		"configuration": {}
	}`)

	summary, err := parser.ParsePlanJSON(jsonData, "test", false)
	if err != nil {
		t.Fatalf("ParsePlanJSON failed: %v", err)
	}

	if summary.ToAdd != 0 || summary.ToChange != 0 || summary.ToDestroy != 0 {
		t.Errorf("Expected no changes, got add=%d change=%d destroy=%d", summary.ToAdd, summary.ToChange, summary.ToDestroy)
	}
}

// TestParseInvalidJSON tests error handling for invalid JSON.
func TestParseInvalidJSON(t *testing.T) {
	jsonData := []byte(`{invalid json}`)

	_, err := parser.ParsePlanJSON(jsonData, "test", false)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

// TestParseDestroyPlanJSON tests parsing with destroy flag.
func TestParseDestroyPlanJSON(t *testing.T) {
	jsonData := []byte(`{
		"format_version": "1.2",
		"terraform_version": "1.5.0",
		"planned_values": {},
		"resource_changes": [
			{
				"address": "aws_instance.example",
				"mode": "managed",
				"type": "aws_instance",
				"name": "example",
				"provider_name": "registry.terraform.io/hashicorp/aws",
				"change": {
					"actions": ["delete"],
					"before": {"ami": "ami-12345678"},
					"after": null,
					"after_unknown": {},
					"before_sensitive": false,
					"after_sensitive": {}
				}
			}
		],
		"output_changes": {},
		"resource_drift": null,
		"configuration": {}
	}`)

	summary, err := parser.ParsePlanJSON(jsonData, "prod", true)
	if err != nil {
		t.Fatalf("ParsePlanJSON failed: %v", err)
	}

	if !summary.IsDestroyPlan {
		t.Error("Expected IsDestroyPlan to be true")
	}

	if summary.Workspace != "prod" {
		t.Errorf("Expected workspace 'prod', got '%s'", summary.Workspace)
	}
}

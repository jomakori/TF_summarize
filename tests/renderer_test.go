package tests

import (
	"strings"
	"testing"

	"github.com/jomakori/TF_summarize/internal"
	"github.com/jomakori/TF_summarize/internal/render"
)

func TestRenderPlanCreate(t *testing.T) {
	s := &internal.Summary{
		Phase:     internal.PhasePlan,
		Workspace: "plat-ue2-sandbox",
		ToAdd:     3,
		RawOutput: "some plan output",
		Creates: []internal.ResourceChange{
			{Address: "module.s3_bucket.aws_s3_bucket.default[0]", Action: internal.ActionCreate},
			{Address: "module.s3_bucket.aws_s3_bucket_acl.default[0]", Action: internal.ActionCreate},
			{Address: "module.s3_bucket.aws_s3_bucket_versioning.default[0]", Action: internal.ActionCreate},
		},
	}

	out := render.Render(s)

	assertContains(t, out, "Terraform Plan")
	assertContains(t, out, "**3** to add")
	assertContains(t, out, "module.s3_bucket.aws_s3_bucket.default[0]")
	assertContains(t, out, "Terraform Plan Output")
}

func TestRenderPlanDestroy(t *testing.T) {
	s := &internal.Summary{
		Phase:     internal.PhasePlan,
		Workspace: "plat-ue2-sandbox",
		ToDestroy: 2,
		ToAdd:     1,
		ToChange:  1,
		Replaces: []internal.ResourceChange{
			{Address: "module.ecs.aws_ecs_task_definition.app", Action: internal.ActionReplace},
		},
		Updates: []internal.ResourceChange{
			{Address: "module.ecs.aws_ecs_service.app", Action: internal.ActionUpdate},
		},
		Destroys: []internal.ResourceChange{
			{Address: "module.ecs.aws_security_group.lb", Action: internal.ActionDestroy},
		},
	}

	out := render.Render(s)

	assertContains(t, out, "Terraform Plan")
	assertContains(t, out, "CAUTION")
}

func TestRenderApplySuccess(t *testing.T) {
	s := &internal.Summary{
		Phase:          internal.PhaseApply,
		Workspace:      "prod",
		ApplySucceeded: true,
		ToAdd:          2,
		RawOutput:      "apply output here",
		Creates: []internal.ResourceChange{
			{Address: "module.s3_bucket.aws_s3_bucket.default[0]", Action: internal.ActionCreate, Success: true},
			{Address: "module.s3_bucket.aws_s3_bucket_acl.default[0]", Action: internal.ActionCreate, Success: true},
		},
	}

	out := render.Render(s)

	assertContains(t, out, "Terraform Apply")
	assertContains(t, out, "✅ Created")
	assertContains(t, out, "module.s3_bucket.aws_s3_bucket.default[0]")
	assertContains(t, out, "Terraform Apply Output")
}

func TestRenderApplyMixed(t *testing.T) {
	s := &internal.Summary{
		Phase:     internal.PhaseApply,
		Workspace: "prod",
		ToAdd:     1,
		Creates: []internal.ResourceChange{
			{Address: "module.s3_bucket.aws_s3_bucket.default[0]", Action: internal.ActionCreate, Success: true},
		},
		Failures: []internal.ResourceChange{
			{Address: "module.rds.aws_db_instance.main", Action: internal.ActionCreate, Error: "creating RDS DB Instance (mydb): DBInstanceAlreadyExists"},
		},
		Errors: []string{"creating RDS DB Instance (mydb): DBInstanceAlreadyExists"},
	}

	out := render.Render(s)

	assertContains(t, out, "Terraform Apply")
	assertContains(t, out, "✅ Created")
	assertContains(t, out, "❌ Failed")
	assertContains(t, out, "module.rds.aws_db_instance.main")
	assertContains(t, out, "DBInstanceAlreadyExists")
}

func TestRenderApplyFail(t *testing.T) {
	s := &internal.Summary{
		Phase:     internal.PhaseApply,
		Workspace: "prod",
		Failures: []internal.ResourceChange{
			{Address: "module.rds.aws_db_instance.main", Action: internal.ActionCreate, Error: "creating RDS DB Instance (mydb): DBInstanceAlreadyExists"},
		},
		Errors: []string{"creating RDS DB Instance (mydb): DBInstanceAlreadyExists"},
	}

	out := render.Render(s)

	assertContains(t, out, "Terraform Apply")
	assertContains(t, out, "❌ Failed")
	assertContains(t, out, "module.rds.aws_db_instance.main")
}

func TestRenderApplyWithDestroys(t *testing.T) {
	s := &internal.Summary{
		Phase:          internal.PhaseApply,
		Workspace:      "staging",
		ApplySucceeded: true,
		ToDestroy:      1,
		ToChange:       1,
		Destroys: []internal.ResourceChange{
			{Address: "module.s3_bucket.aws_s3_bucket.default[0]", Action: internal.ActionDestroy, Success: true},
		},
		Updates: []internal.ResourceChange{
			{Address: "module.vpc.aws_subnet.private[0]", Action: internal.ActionUpdate, Success: true},
		},
	}

	out := render.Render(s)

	assertContains(t, out, "Terraform Apply")
	assertContains(t, out, "✅ Destroyed")
	assertContains(t, out, "✅ Updated")
}

func TestRenderNoChanges(t *testing.T) {
	s := &internal.Summary{
		Phase:     internal.PhasePlan,
		Workspace: "dev",
	}

	out := render.Render(s)
	assertContains(t, out, "Terraform Plan")
	assertContains(t, out, "Infrastructure is up-to-date")
}

func TestRenderRawOutputFormatting(t *testing.T) {
	// Test that raw output is properly formatted with color codes
	rawOutput := `  # module.s3_bucket.aws_s3_bucket.default[0] will be created
  + resource "aws_s3_bucket" "default" {
      + bucket = "my-test-bucket"
    }

  # module.rds.aws_db_instance.main will be destroyed
  - resource "aws_db_instance" "main" {
      - engine = "postgres"
    }

  # module.vpc.aws_subnet.private[0] will be updated
  ~ resource "aws_subnet" "private" {
      ~ tags = {...}
    }

Plan: 1 to add, 1 to change, 1 to destroy.`

	s := &internal.Summary{
		Phase:     internal.PhasePlan,
		Workspace: "test",
		ToAdd:     1,
		ToChange:  1,
		ToDestroy: 1,
		RawOutput: rawOutput,
	}

	out := render.Render(s)

	// Verify output contains the raw output section
	assertContains(t, out, "Terraform Plan Output")
	assertContains(t, out, "<details>")
	assertContains(t, out, "```diff")
	
	// Verify color codes are applied (diff markers should be present)
	assertContains(t, out, "+ resource")
	assertContains(t, out, "- resource")
	assertContains(t, out, "! resource")
	
	// Verify plan summary is included
	assertContains(t, out, "Plan: 1 to add, 1 to change, 1 to destroy")
}

func TestRenderApplyOutputFormatting(t *testing.T) {
	// Test that apply output is properly formatted with color codes
	rawOutput := `module.s3_bucket.aws_s3_bucket.default[0]: Creating...
module.s3_bucket.aws_s3_bucket.default[0]: Creation complete after 2s [id=my-test-bucket]
module.rds.aws_db_instance.main: Destroying... [id=mydb]
module.rds.aws_db_instance.main: Destruction complete after 5s

Apply complete! Resources: 1 added, 0 changed, 1 destroyed.`

	s := &internal.Summary{
		Phase:          internal.PhaseApply,
		Workspace:      "prod",
		ToAdd:          1,
		ToDestroy:      1,
		ApplySucceeded: true,
		RawOutput:      rawOutput,
	}

	out := render.Render(s)

	// Verify output contains the apply output section
	assertContains(t, out, "Terraform Apply Output")
	assertContains(t, out, "<details>")
	assertContains(t, out, "```diff")
	
	// Verify color codes are applied
	assertContains(t, out, "+ module.s3_bucket")
	assertContains(t, out, "- module.rds")
	
	// Verify apply complete message is included
	assertContains(t, out, "Apply complete!")
}

// Issue 2: Test that "No Changes" badge is NOT shown when apply-phase changes exist
func TestNoChangesBadgeWithApplyChanges(t *testing.T) {
	tests := []struct {
		name           string
		summary        *internal.Summary
		expectNoChange bool
	}{
		{
			name: "No changes badge shown when truly no changes",
			summary: &internal.Summary{
				Phase:     internal.PhasePlan,
				Workspace: "test",
				// All counts are zero, no resources
			},
			expectNoChange: true,
		},
		{
			name: "No changes badge NOT shown when Creates exist",
			summary: &internal.Summary{
				Phase:     internal.PhaseApply,
				Workspace: "test",
				Creates: []internal.ResourceChange{
					{Address: "aws_s3_bucket.test", Action: internal.ActionCreate, Success: true},
				},
			},
			expectNoChange: false,
		},
		{
			name: "No changes badge NOT shown when Destroys exist",
			summary: &internal.Summary{
				Phase:     internal.PhaseApply,
				Workspace: "test",
				Destroys: []internal.ResourceChange{
					{Address: "aws_s3_bucket.test", Action: internal.ActionDestroy, Success: true},
				},
			},
			expectNoChange: false,
		},
		{
			name: "No changes badge NOT shown when Updates exist",
			summary: &internal.Summary{
				Phase:     internal.PhaseApply,
				Workspace: "test",
				Updates: []internal.ResourceChange{
					{Address: "aws_s3_bucket.test", Action: internal.ActionUpdate, Success: true},
				},
			},
			expectNoChange: false,
		},
		{
			name: "No changes badge NOT shown when Failures exist",
			summary: &internal.Summary{
				Phase:     internal.PhaseApply,
				Workspace: "test",
				Failures: []internal.ResourceChange{
					{Address: "aws_s3_bucket.test", Action: internal.ActionCreate, Error: "failed"},
				},
			},
			expectNoChange: false,
		},
		{
			name: "No changes badge NOT shown when ToAdd > 0",
			summary: &internal.Summary{
				Phase:     internal.PhasePlan,
				Workspace: "test",
				ToAdd:     1,
			},
			expectNoChange: false,
		},
		{
			name: "No changes badge NOT shown when ToChange > 0",
			summary: &internal.Summary{
				Phase:     internal.PhasePlan,
				Workspace: "test",
				ToChange:  1,
			},
			expectNoChange: false,
		},
		{
			name: "No changes badge NOT shown when ToDestroy > 0",
			summary: &internal.Summary{
				Phase:     internal.PhasePlan,
				Workspace: "test",
				ToDestroy: 1,
			},
			expectNoChange: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := render.Render(tc.summary)
			hasNoChangeBadge := strings.Contains(out, "No%20Changes") || strings.Contains(out, "No Changes")

			if tc.expectNoChange && !hasNoChangeBadge {
				t.Errorf("expected 'No Changes' badge but it was not found.\nOutput:\n%s", out)
			}
			if !tc.expectNoChange && hasNoChangeBadge {
				t.Errorf("did NOT expect 'No Changes' badge but it was found.\nOutput:\n%s", out)
			}
		})
	}
}

// Test comprehensive plan with drift detection and modify section
func TestPlanWithDriftAndModify(t *testing.T) {
	s := &internal.Summary{
		Phase:         internal.PhasePlan,
		Workspace:     "plat-ue2-sandbox",
		ToAdd:         8,
		ToChange:      1,
		DriftDetected: true,
		Creates: []internal.ResourceChange{
			{Address: "tailscale_tailnet_key.vm_auth_key", Action: internal.ActionCreate},
			{Address: "module.logging.oci_logging_log_group.linuxloggroup", Action: internal.ActionCreate},
			{Address: "module.vm.oci_core_instance.instance[0]", Action: internal.ActionCreate},
			{Address: "module.vm.oci_core_volume_attachment.instance_volume_attachment[0]", Action: internal.ActionCreate},
			{Address: "module.vm.oci_core_volume.boot_volume[0]", Action: internal.ActionCreate},
			{Address: "module.vm.oci_core_vnic_attachment.instance_vnic_attachment[0]", Action: internal.ActionCreate},
			{Address: "module.vm.oci_core_instance_configuration.instance_pool[0]", Action: internal.ActionCreate},
			{Address: "module.vm.oci_core_instance_pool.instance_pool[0]", Action: internal.ActionCreate},
		},
		Updates: []internal.ResourceChange{
			{Address: "doppler_secret.tailscale_auth_key", Action: internal.ActionUpdate},
		},
		Errors: []string{
			"Invalid function argument on outputs.tf line 6",
		},
		Outputs: []internal.OutputChange{
			{Name: "verification_commands", Action: internal.ActionCreate, Value: "(known after apply)"},
			{Name: "vm_instance_id", Action: internal.ActionCreate, Value: "(known after apply)"},
			{Name: "vm_private_ip", Action: internal.ActionCreate, Value: "(known after apply)"},
			{Name: "vm_public_ip", Action: internal.ActionCreate, Value: "(known after apply)"},
		},
		RawOutput: `Plan: 8 to add, 1 to change, 0 to destroy.

# tailscale_tailnet_key.vm_auth_key has been deleted
# doppler_secret.tailscale_auth_key will be updated in-place
# (and 7 more resources to create)`,
	}

	out := render.Render(s)

	// Verify badges (URL-encoded in image URLs)
	assertContains(t, out, "Create%20%288%29")
	assertContains(t, out, "Modify%20%281%29")
	assertContains(t, out, "Drift%20Detected")

	// Verify summary line
	assertContains(t, out, "**8** to add")
	assertContains(t, out, "**1** to change")

	// Verify drift warning uses markdown alert (not GHA command)
	assertContains(t, out, "> [!WARNING]")
	assertContains(t, out, "**Drift detected!** Objects have changed outside of Terraform.")
	// Ensure no GHA workflow commands
	assertNotContains(t, out, "::warning::")

	// Verify error message
	assertContains(t, out, "> [!ERROR]")
	assertContains(t, out, "Invalid function argument on outputs.tf line 6")

	// Verify modify section uses ! prefix (standard diff notation for modifications)
	assertContains(t, out, "! doppler_secret.tailscale_auth_key")

	// Verify create resources
	assertContains(t, out, "+ tailscale_tailnet_key.vm_auth_key")
	assertContains(t, out, "+ module.logging.oci_logging_log_group.linuxloggroup")

	// Verify outputs section with bold count
	assertContains(t, out, "<summary><b>Outputs</b> <b>(4)</b></summary>")
	assertContains(t, out, "+ verification_commands = (known after apply)")

	// Verify raw output section
	assertContains(t, out, "Terraform Plan Output")
}

// Issue 3A: Test that failed resources use ERROR callout for red highlighting
func TestFailedResourcesUseErrorCallout(t *testing.T) {
	s := &internal.Summary{
		Phase:     internal.PhaseApply,
		Workspace: "test",
		Failures: []internal.ResourceChange{
			{Address: "provider_api_key.auth_key", Action: internal.ActionCreate, Error: "Failed to create key"},
		},
	}

	out := render.Render(s)

	// Should contain the ERROR callout for red highlighting (not CAUTION - that's for warnings)
	assertContains(t, out, "> [!ERROR]")
	assertContains(t, out, "Failed to create key")
	assertContains(t, out, "provider_api_key.auth_key")
}

// Test that generic exit code messages are filtered out from error display
func TestGenericExitCodeMessagesFiltered(t *testing.T) {
	s := &internal.Summary{
		Phase:     internal.PhasePlan,
		Workspace: "test",
		Errors: []string{
			"provider credentials are empty - set api_key",
			"Terraform exited with code 1.",
			"Terraform exited with code 2",
		},
	}

	out := render.Render(s)

	// Real errors should be displayed with ERROR callout
	assertContains(t, out, "> [!ERROR]")
	assertContains(t, out, "provider credentials are empty")

	// Generic exit code messages should be filtered out
	assertNotContains(t, out, "exited with code 1")
	assertNotContains(t, out, "exited with code 2")
}

// Test that errors use [!ERROR] callout, NOT [!CAUTION]
func TestErrorsUseErrorCalloutNotCaution(t *testing.T) {
	s := &internal.Summary{
		Phase:     internal.PhasePlan,
		Workspace: "test",
		ToAdd:     6,
		Errors: []string{
			"Unable to determine network device. Exiting.",
			"Tailscale installation script failed. Retry attempt 1",
			"Tailscale installation could not be verified. Retry attempt 1",
		},
		Creates: []internal.ResourceChange{
			{Address: "provider_secret.auth_key", Action: internal.ActionCreate},
			{Address: "provider_api_key.vm_auth_key", Action: internal.ActionCreate},
		},
	}

	out := render.Render(s)

	// Errors should use [!ERROR] callout
	assertContains(t, out, "> [!ERROR]")
	assertContains(t, out, "Unable to determine network device")
	assertContains(t, out, "Tailscale installation script failed")
	assertContains(t, out, "Tailscale installation could not be verified")

	// Count occurrences of [!ERROR] - should be 3 (one for each error)
	errorCount := strings.Count(out, "> [!ERROR]")
	if errorCount != 3 {
		t.Errorf("expected 3 [!ERROR] callouts, got %d", errorCount)
	}

	// [!CAUTION] should NOT be used for errors (only for warnings like "will delete resources")
	// Since we have no destroys, there should be no [!CAUTION]
	cautionCount := strings.Count(out, "> [!CAUTION]")
	if cautionCount != 0 {
		t.Errorf("expected 0 [!CAUTION] callouts for errors, got %d.\nOutput:\n%s", cautionCount, out)
	}

	// Print output for visual inspection
	t.Logf("Rendered output:\n%s", out)
}

// Issue 3B: Test that error blocks in raw output are highlighted in red
func TestErrorBlocksHighlightedInRed(t *testing.T) {
	// Simulate Terraform error output with error block markers
	rawOutput := `module.network.provider_vpc.main: Creating...
module.network.provider_vpc.main: Creation complete after 1s [id=vpc-test123]
╷
│ Error: Invalid index
│
│   on main.tf line 91, in module "compute":
│   91:   subnet_ids = module.network.subnet_id["nonexistent-subnet"]
│
│ The given key does not identify an element in this collection value.
╵
╷
│ Error: Failed to create resource
│
│   with provider_api_key.auth_key,
│   on main.tf line 108, in resource "provider_api_key" "auth_key":
│  108: resource "provider_api_key" "auth_key" {
│
│ API token invalid (401)
╵`

	s := &internal.Summary{
		Phase:     internal.PhaseApply,
		Workspace: "test",
		RawOutput: rawOutput,
		Failures: []internal.ResourceChange{
			{Address: "provider_api_key.auth_key", Action: internal.ActionCreate, Error: "Failed to create resource"},
		},
	}

	out := render.Render(s)

	// Error block start marker should be prefixed with - for red highlighting
	assertContains(t, out, "- ╷")
	// Error block end marker should be prefixed with - for red highlighting
	assertContains(t, out, "- ╵")
	// Lines inside error block should be prefixed with - for red highlighting
	assertContains(t, out, "- │ Error:")
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected output to contain %q, but it didn't.\nOutput:\n%s", needle, haystack)
	}
}

func assertNotContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Errorf("expected output to NOT contain %q, but it did.\nOutput:\n%s", needle, haystack)
	}
}

// Test comprehensive error highlighting with realistic Terraform apply output
func TestErrorHighlightingWithRealOutput(t *testing.T) {
	// Simulates a Terraform apply with multiple resources created successfully
	// and two different types of errors:
	// 1. Configuration error (Invalid index) - no specific resource
	// 2. Resource creation error (API failure) - with specific resource
	rawOutput := `+ provider_api_key.auth_key: Creating...
+ module.network.provider_vpc.main: Creating...
+ module.network.provider_vpc.main: Creation complete after 1s [id=vpc-abc123]
module.network.module.subnet[0].data.provider_dhcp_options.options: Reading...
+ module.network.provider_security_list.default[0]: Creating...
+ module.network.provider_internet_gateway.main[0]: Creating...
+ provider_security_list.public: Creating...
module.network.module.subnet[0].data.provider_dhcp_options.options: Read complete after 0s [id=dhcp-123]
+ module.network.provider_security_list.default[0]: Creation complete after 0s [id=seclist-abc]
+ provider_security_list.public: Creation complete after 0s [id=seclist-def]
module.compute.data.provider_policies.backup: Reading...
module.compute.data.provider_availability_domains.ad: Reading...
+ module.network.provider_internet_gateway.main[0]: Creation complete after 1s [id=igw-abc]
+ module.network.provider_route_table.main[0]: Creating...
module.compute.data.provider_policies.backup: Read complete after 1s [id=policy-0]
module.compute.data.provider_availability_domains.ad: Read complete after 1s [id=ad-123]
module.compute.data.provider_shapes.current: Reading...
+ module.network.provider_route_table.main[0]: Creation complete after 0s [id=rt-abc]
+ module.network.module.subnet[0].provider_subnet.main["public"]: Creating...
module.compute.data.provider_shapes.current: Read complete after 0s [id=shapes-123]
+ module.network.module.subnet[0].provider_subnet.main["public"]: Creation complete after 2s [id=subnet-abc]
╷
│ Error: Invalid index
│
│   on main.tf line 91, in module "compute":
│   91:   subnet_ids = module.network.subnet_id["nonexistent-subnet"]
│     ├────────────────
│     │ module.network.subnet_id is object with 1 attribute "public"
│     │ var.name is "my-project"
│
│ The given key does not identify an element in this collection value.
╵
╷
│ Error: Failed to create resource
│
│   with provider_api_key.auth_key,
│   on main.tf line 108, in resource "provider_api_key" "auth_key":
│  108: resource "provider_api_key" "auth_key" {
│
│ API token invalid (401)
╵
::error::Terraform exited with code 1`

	s := &internal.Summary{
		Phase:     internal.PhaseApply,
		Workspace: "test",
		RawOutput: rawOutput,
		Creates: []internal.ResourceChange{
			{Address: "module.network.provider_vpc.main", Action: internal.ActionCreate, Success: true},
			{Address: "module.network.provider_security_list.default[0]", Action: internal.ActionCreate, Success: true},
			{Address: "provider_security_list.public", Action: internal.ActionCreate, Success: true},
			{Address: "module.network.provider_internet_gateway.main[0]", Action: internal.ActionCreate, Success: true},
			{Address: "module.network.provider_route_table.main[0]", Action: internal.ActionCreate, Success: true},
			{Address: "module.network.module.subnet[0].provider_subnet.main[\"public\"]", Action: internal.ActionCreate, Success: true},
		},
		Failures: []internal.ResourceChange{
			{Address: "provider_api_key.auth_key", Action: internal.ActionCreate, Error: "Failed to create resource"},
		},
	}

	out := render.Render(s)

	// Test 1: Error block start markers should be highlighted in red (prefixed with -)
	assertContains(t, out, "- ╷")

	// Test 2: Error block end markers should be highlighted in red (prefixed with -)
	assertContains(t, out, "- ╵")

	// Test 3: Error lines inside blocks should be highlighted in red
	assertContains(t, out, "- │ Error: Invalid index")
	assertContains(t, out, "- │ Error: Failed to create resource")

	// Test 4: Lines inside error blocks should be highlighted in red
	assertContains(t, out, "- │   on main.tf line 91")
	assertContains(t, out, "- │ API token invalid (401)")

	// Test 5: Failed resources section should use ERROR callout (not CAUTION - that's for warnings)
	assertContains(t, out, "> [!ERROR]")
	assertContains(t, out, "provider_api_key.auth_key")

	// Test 6: No Changes badge should NOT appear (we have creates and failures)
	assertNotContains(t, out, "No%20Changes")

	// Test 7: Failed badge should appear
	assertContains(t, out, "Failed")

	// Test 8: Creation lines should be highlighted in green (prefixed with +)
	assertContains(t, out, "+ module.network.provider_vpc.main: Creating...")
	assertContains(t, out, "+ module.network.provider_vpc.main: Creation complete")

	// Print the output for visual inspection
	t.Logf("Rendered output:\n%s", out)
}

// Test that Terraform outputs are rendered in a separate section
func TestOutputsSection(t *testing.T) {
	s := &internal.Summary{
		Phase:     internal.PhasePlan,
		Workspace: "test",
		ToAdd:     2,
		Creates: []internal.ResourceChange{
			{Address: "aws_instance.web", Action: internal.ActionCreate},
			{Address: "aws_s3_bucket.data", Action: internal.ActionCreate},
		},
		Outputs: []internal.OutputChange{
			{Name: "instance_id", Action: internal.ActionCreate, Value: "(known after apply)"},
			{Name: "public_ip", Action: internal.ActionCreate, Value: "(known after apply)"},
			{Name: "bucket_name", Action: internal.ActionCreate, Value: "my-bucket"},
			{Name: "old_output", Action: internal.ActionDestroy, Value: "old-value"},
		},
	}

	out := render.Render(s)

	// Should have an Outputs section with bold count
	assertContains(t, out, "<summary><b>Outputs</b> <b>(4)</b></summary>")

	// Should show outputs with diff syntax
	assertContains(t, out, "+ instance_id = (known after apply)")
	assertContains(t, out, "+ public_ip = (known after apply)")
	assertContains(t, out, "+ bucket_name = my-bucket")
	assertContains(t, out, "- old_output = old-value")

	// Print output for visual inspection
	t.Logf("Rendered output:\n%s", out)
}

// Test that outputs section is not shown when there are no outputs
func TestNoOutputsSection(t *testing.T) {
	s := &internal.Summary{
		Phase:     internal.PhasePlan,
		Workspace: "test",
		ToAdd:     1,
		Creates: []internal.ResourceChange{
			{Address: "aws_instance.web", Action: internal.ActionCreate},
		},
		// No outputs
	}

	out := render.Render(s)

	// Should NOT have an Outputs section
	assertNotContains(t, out, "<summary><b>Outputs</b>")
}

// Test that error formatting is provider-dependent
func TestErrorFormattingProviderDependent(t *testing.T) {
	tests := []struct {
		name           string
		targetProvider internal.OutputTarget
	}{
		{
			name:           "GHA provider outputs markdown alerts only",
			targetProvider: internal.TargetGHASummary,
		},
		{
			name:           "PR provider outputs markdown alerts only",
			targetProvider: internal.TargetPR,
		},
		{
			name:           "Stdout provider outputs markdown alerts only",
			targetProvider: internal.TargetStdout,
		},
		{
			name:           "Empty/default provider outputs markdown alerts only",
			targetProvider: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &internal.Summary{
				Phase:          internal.PhasePlan,
				Workspace:      "test",
				TargetProvider: tc.targetProvider,
				Errors: []string{
					"Test error message",
				},
			}

			out := render.Render(s)

			// All providers should have the markdown alert
			assertContains(t, out, "> [!ERROR]")
			assertContains(t, out, "Test error message")

			// No providers should have ::error:: commands (GHA commands removed)
			hasGHAError := strings.Contains(out, "::error::")
			if hasGHAError {
				t.Errorf("did NOT expect ::error:: command for provider %q but it was found.\nOutput:\n%s", tc.targetProvider, out)
			}
		})
	}
}

// Test that apply failures also respect provider-dependent formatting
func TestApplyFailureFormattingProviderDependent(t *testing.T) {
	tests := []struct {
		name           string
		targetProvider internal.OutputTarget
	}{
		{
			name:           "GHA provider outputs markdown alerts only for failures",
			targetProvider: internal.TargetGHASummary,
		},
		{
			name:           "Stdout provider outputs markdown alerts only for failures",
			targetProvider: internal.TargetStdout,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &internal.Summary{
				Phase:          internal.PhaseApply,
				Workspace:      "test",
				TargetProvider: tc.targetProvider,
				Failures: []internal.ResourceChange{
					{Address: "aws_instance.test", Action: internal.ActionCreate, Error: "Instance creation failed"},
				},
			}

			out := render.Render(s)

			// All providers should have the markdown alert
			assertContains(t, out, "> [!ERROR]")
			assertContains(t, out, "Instance creation failed")

			// No providers should have ::error:: commands (GHA commands removed)
			hasGHAError := strings.Contains(out, "::error::")
			if hasGHAError {
				t.Errorf("did NOT expect ::error:: command for provider %q but it was found.\nOutput:\n%s", tc.targetProvider, out)
			}
		})
	}
}

// Test that warning formatting is provider-dependent
func TestWarningFormattingProviderDependent(t *testing.T) {
	tests := []struct {
		name           string
		targetProvider internal.OutputTarget
	}{
		{
			name:           "GHA provider outputs markdown alerts only",
			targetProvider: internal.TargetGHASummary,
		},
		{
			name:           "PR provider outputs markdown alerts only",
			targetProvider: internal.TargetPR,
		},
		{
			name:           "Stdout provider outputs markdown alerts only",
			targetProvider: internal.TargetStdout,
		},
		{
			name:           "Empty/default provider outputs markdown alerts only",
			targetProvider: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &internal.Summary{
				Phase:          internal.PhasePlan,
				Workspace:      "test",
				TargetProvider: tc.targetProvider,
				Warnings: []string{
					"Test warning message",
				},
			}

			out := render.Render(s)

			// All providers should have the markdown alert
			assertContains(t, out, "> [!WARNING]")
			assertContains(t, out, "Test warning message")

			// No providers should have ::warning:: commands (GHA commands removed)
			hasGHAWarning := strings.Contains(out, "::warning::")
			if hasGHAWarning {
				t.Errorf("did NOT expect ::warning:: command for provider %q but it was found.\nOutput:\n%s", tc.targetProvider, out)
			}
		})
	}
}

// Test that caution messages (destructive operations) are provider-dependent
func TestCautionFormattingProviderDependent(t *testing.T) {
	tests := []struct {
		name           string
		targetProvider internal.OutputTarget
	}{
		{
			name:           "GHA provider outputs markdown alerts only for caution",
			targetProvider: internal.TargetGHASummary,
		},
		{
			name:           "PR provider outputs markdown alerts only for caution",
			targetProvider: internal.TargetPR,
		},
		{
			name:           "Stdout provider outputs markdown alerts only for caution",
			targetProvider: internal.TargetStdout,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &internal.Summary{
				Phase:          internal.PhasePlan,
				Workspace:      "test",
				TargetProvider: tc.targetProvider,
				ToDestroy:      1,
				Destroys: []internal.ResourceChange{
					{Address: "aws_instance.test", Action: internal.ActionDestroy},
				},
			}

			out := render.Render(s)

			// All providers should have the markdown CAUTION alert
			assertContains(t, out, "> [!CAUTION]")
			assertContains(t, out, "Terraform will delete resources")

			// No providers should have ::warning:: commands (GHA commands removed)
			hasGHAWarning := strings.Contains(out, "::warning::")
			if hasGHAWarning {
				t.Errorf("did NOT expect ::warning:: command for provider %q but it was found.\nOutput:\n%s", tc.targetProvider, out)
			}
		})
	}
}

// Test that drift warnings are provider-dependent
func TestDriftWarningFormattingProviderDependent(t *testing.T) {
	tests := []struct {
		name           string
		targetProvider internal.OutputTarget
	}{
		{
			name:           "GHA provider outputs markdown alerts only for drift",
			targetProvider: internal.TargetGHASummary,
		},
		{
			name:           "Stdout provider outputs markdown alerts only for drift",
			targetProvider: internal.TargetStdout,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &internal.Summary{
				Phase:          internal.PhasePlan,
				Workspace:      "test",
				TargetProvider: tc.targetProvider,
				DriftDetected:  true,
			}

			out := render.Render(s)

			// All providers should have the markdown WARNING alert
			assertContains(t, out, "> [!WARNING]")
			assertContains(t, out, "Drift detected")

			// No providers should have ::warning:: commands (GHA commands removed)
			hasGHAWarning := strings.Contains(out, "::warning::")
			if hasGHAWarning {
				t.Errorf("did NOT expect ::warning:: command for provider %q but it was found.\nOutput:\n%s", tc.targetProvider, out)
			}
		})
	}
}

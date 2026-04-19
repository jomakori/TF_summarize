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
	assertContains(t, out, "~ resource")
	
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

// Issue 3A: Test that failed resources use CAUTION callout for red highlighting
func TestFailedResourcesUseCautionCallout(t *testing.T) {
	s := &internal.Summary{
		Phase:     internal.PhaseApply,
		Workspace: "test",
		Failures: []internal.ResourceChange{
			{Address: "tailscale_tailnet_key.vm_auth_key", Action: internal.ActionCreate, Error: "Failed to create key"},
		},
	}

	out := render.Render(s)

	// Should contain the CAUTION callout for red highlighting
	assertContains(t, out, "> [!CAUTION]")
	assertContains(t, out, "Failed to create key")
	assertContains(t, out, "tailscale_tailnet_key.vm_auth_key")
}

// Issue 3B: Test that error blocks in raw output are highlighted in red
func TestErrorBlocksHighlightedInRed(t *testing.T) {
	// Simulate Terraform error output with error block markers
	rawOutput := `module.vcn.oci_core_vcn.vcn: Creating...
module.vcn.oci_core_vcn.vcn: Creation complete after 1s [id=ocid1.vcn.oc1.iad.test]
╷
│ Error: Invalid index
│
│   on 1-vm.tf line 91, in module "compute_instance":
│   91:   subnet_ocids = module.vcn.subnet_id["test-public-subnet"]
│
│ The given key does not identify an element in this collection value.
╵
╷
│ Error: Failed to create key
│
│   with tailscale_tailnet_key.vm_auth_key,
│   on 1-vm.tf line 108, in resource "tailscale_tailnet_key" "vm_auth_key":
│  108: resource "tailscale_tailnet_key" "vm_auth_key" {
│
│ API token invalid (401)
╵`

	s := &internal.Summary{
		Phase:     internal.PhaseApply,
		Workspace: "test",
		RawOutput: rawOutput,
		Failures: []internal.ResourceChange{
			{Address: "tailscale_tailnet_key.vm_auth_key", Action: internal.ActionCreate, Error: "Failed to create key"},
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

	// Test 5: Failed resources section should use CAUTION callout
	assertContains(t, out, "> [!CAUTION]")
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

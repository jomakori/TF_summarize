package tests

import (
	"strings"
	"testing"

	"github.com/jomakori/TF_summarize/internal"
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

	out := internal.Render(s)

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

	out := internal.Render(s)

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

	out := internal.Render(s)

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

	out := internal.Render(s)

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

	out := internal.Render(s)

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

	out := internal.Render(s)

	assertContains(t, out, "Terraform Apply")
	assertContains(t, out, "✅ Destroyed")
	assertContains(t, out, "✅ Updated")
}

func TestRenderNoChanges(t *testing.T) {
	s := &internal.Summary{
		Phase:     internal.PhasePlan,
		Workspace: "dev",
	}

	out := internal.Render(s)
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

	out := internal.Render(s)

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

	out := internal.Render(s)

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

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected output to contain %q, but it didn't.\nOutput:\n%s", needle, haystack)
	}
}

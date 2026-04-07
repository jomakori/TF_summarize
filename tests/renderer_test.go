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

	assertContains(t, out, "Changes found for")
	assertContains(t, out, "plat-ue2-sandbox")
	assertContains(t, out, "-Create")
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

	assertContains(t, out, "CAUTION")
	assertContains(t, out, "-Replace")
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

	assertContains(t, out, "✅")
	assertContains(t, out, "applied successfully")
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

	assertContains(t, out, "❌")
	assertContains(t, out, "Apply failed")
	assertContains(t, out, "✅ Created")
	assertContains(t, out, "❌ Failed")
	assertContains(t, out, "module.rds.aws_db_instance.main")
	assertContains(t, out, "DBInstanceAlreadyExists")
	assertContains(t, out, "-Failed")
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

	assertContains(t, out, "❌")
	assertContains(t, out, "Apply failed")
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

	assertContains(t, out, "✅")
	assertContains(t, out, "applied successfully")
	assertContains(t, out, "✅ Destroyed")
	assertContains(t, out, "✅ Updated")
	assertContains(t, out, "**1** destroyed")
	assertContains(t, out, "**1** changed")
}

func TestRenderNoChanges(t *testing.T) {
	s := &internal.Summary{
		Phase:     internal.PhasePlan,
		Workspace: "dev",
	}

	out := internal.Render(s)
	assertContains(t, out, "No changes found")
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected output to contain %q, but it didn't.\nOutput:\n%s", needle, haystack)
	}
}

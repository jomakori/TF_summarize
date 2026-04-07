 package tests

import (
	"testing"

	"github.com/jomakori/TF_summarize/internal"
)

const planCreateOutput = `
Terraform used the selected providers to generate the following execution plan.
Resource actions are indicated with the following symbols:
  + create

Terraform will perform the following actions:

  # module.s3_bucket.aws_s3_bucket.default[0] will be created
  + resource "aws_s3_bucket" "default" {
      + arn            = (known after apply)
      + bucket         = "my-test-bucket"
      + id             = (known after apply)
    }

  # module.s3_bucket.aws_s3_bucket_acl.default[0] will be created
  + resource "aws_s3_bucket_acl" "default" {
      + bucket = (known after apply)
      + id     = (known after apply)
    }

  # module.s3_bucket.aws_s3_bucket_versioning.default[0] will be created
  + resource "aws_s3_bucket_versioning" "default" {
      + bucket = (known after apply)
      + id     = (known after apply)
    }

Plan: 3 to add, 0 to change, 0 to destroy.
`

const planReplaceOutput = `
Terraform used the selected providers to generate the following execution plan.

  # module.ecs.aws_ecs_task_definition.app must be replaced
  -/+ resource "aws_ecs_task_definition" "app" {
        ~ arn       = "arn:aws:ecs:..." -> (known after apply)
        ~ revision  = 5 -> (known after apply)
      }

  # module.ecs.aws_ecs_service.app will be updated in-place
  ~ resource "aws_ecs_service" "app" {
        id   = "arn:aws:ecs:..."
      ~ task_definition = "..." -> (known after apply)
    }

  # module.ecs.aws_security_group.lb will be destroyed
  - resource "aws_security_group" "lb" {
      - arn  = "arn:aws:ec2:..." -> null
      - id   = "sg-123" -> null
    }

Plan: 1 to add, 1 to change, 2 to destroy.

Warning: Applied changes may be incomplete
`

const applySuccessOutput = `
module.s3_bucket.aws_s3_bucket.default[0]: Creating...
module.s3_bucket.aws_s3_bucket.default[0]: Creation complete after 2s [id=my-test-bucket]
module.s3_bucket.aws_s3_bucket_acl.default[0]: Creating...
module.s3_bucket.aws_s3_bucket_acl.default[0]: Creation complete after 1s [id=my-test-bucket,private]

Apply complete! Resources: 2 added, 0 changed, 0 destroyed.
`

const applyMixedOutput = `
module.s3_bucket.aws_s3_bucket.default[0]: Creating...
module.s3_bucket.aws_s3_bucket.default[0]: Creation complete after 2s [id=my-test-bucket]
module.rds.aws_db_instance.main: Creating...

Error: creating RDS DB Instance (mydb): DBInstanceAlreadyExists

  with module.rds.aws_db_instance.main,
  on modules/rds/main.tf line 1, in resource "aws_db_instance" "main":
   1: resource "aws_db_instance" "main" {
`

const applyFailOutput = `
module.rds.aws_db_instance.main: Creating...

Error: creating RDS DB Instance (mydb): DBInstanceAlreadyExists

  with module.rds.aws_db_instance.main,
  on main.tf line 42, in resource "aws_db_instance" "main":
  42: resource "aws_db_instance" "main" {
`

const applyDestroyOutput = `
module.s3_bucket.aws_s3_bucket.default[0]: Destroying... [id=my-test-bucket]
module.s3_bucket.aws_s3_bucket.default[0]: Destruction complete after 3s
module.vpc.aws_subnet.private[0]: Modifying... [id=subnet-abc123]
module.vpc.aws_subnet.private[0]: Modifications complete after 1s [id=subnet-abc123]

Apply complete! Resources: 0 added, 1 changed, 1 destroyed.
`

const noChangesOutput = `
No changes. Your infrastructure matches the configuration.

Terraform has compared your real infrastructure against your configuration
and found no differences, so no changes are needed.
`

const driftOutput = `
Note: Objects have changed outside of Terraform

Terraform detected the following changes made outside of Terraform since the
last "terraform apply" which may have affected this plan:

  # module.vpc.aws_vpc.main has been changed
  ~ resource "aws_vpc" "main" {
        id = "vpc-123"
      ~ tags = {
          + "ManagedBy" = "manual"
        }
    }

Plan: 0 to add, 1 to change, 0 to destroy.
`

const planCreateWithANSIOutput = `
[0m[1mdata.oci_core_images.oracle_linux_arm: Reading...[0m[0m
[0m[1mdata.oci_core_images.oracle_linux_arm: Read complete after 0s [id=CoreImagesDataSource-1239447172][0m

Terraform used the selected providers to generate the following execution
plan. Resource actions are indicated with the following symbols:
  [32m+[0m create
 [36m<=[0m read (data resources)

Terraform will perform the following actions:

[1m  # doppler_secret.tailscale_auth_key[0m will be created
[0m  [32m+[0m[0m resource "doppler_secret" "tailscale_auth_key" {
      [32m+[0m[0m computed   = (sensitive value)
      [32m+[0m[0m config     = "production"
      [32m+[0m[0m id         = (known after apply)
      [32m+[0m[0m name       = "TF_VAR_TAILSCALE_AUTH_KEY"
      [32m+[0m[0m project    = "maklab-base0"
      [32m+[0m[0m value      = (sensitive value)
      [32m+[0m[0m value_type = "string"
      [32m+[0m[0m visibility = "masked"
    }

[1m  # oci_core_security_list.public_security_list[0m will be created
[0m  [32m+[0m[0m resource "oci_core_security_list" "public_security_list" {
      [32m+[0m[0m compartment_id = (sensitive value)
      [32m+[0m[0m display_name   = "maklab-base0-public-security-list"
    }

[1m  # tailscale_tailnet_key.vm_auth_key[0m will be created
[0m  [32m+[0m[0m resource "tailscale_tailnet_key" "vm_auth_key" {
      [32m+[0m[0m created_at    = (known after apply)
      [32m+[0m[0m ephemeral     = false
    }

[1m  # module.compute_instance.data.oci_core_private_ips.private_ips[0][0m will be read during apply
  # (config refers to values not yet known)
[0m [36m<=[0m[0m data "oci_core_private_ips" "private_ips" {
      [32m+[0m[0m id          = (known after apply)
      [32m+[0m[0m private_ips = (known after apply)
      [32m+[0m[0m vnic_id     = (known after apply)
    }

[1m  # module.compute_instance.data.oci_core_shapes.current_ad[0m will be read during apply
  # (config refers to values not yet known)
[0m [36m<=[0m[0m data "oci_core_shapes" "current_ad" {
      [32m+[0m[0m availability_domain = (known after apply)
      [32m+[0m[0m compartment_id      = (sensitive value)
      [32m+[0m[0m id                  = (known after apply)
      [32m+[0m[0m shapes              = (known after apply)
    }

[1mPlan:[0m [0m15 to add, 0 to change, 0 to destroy.
`

func TestParsePlanCreate(t *testing.T) {
	s, err := internal.Parse(planCreateOutput, internal.PhasePlan, "plat-ue2-sandbox", false)
	if err != nil {
		t.Fatal(err)
	}

	if s.ToAdd != 3 {
		t.Errorf("expected 3 to add, got %d", s.ToAdd)
	}
	if s.ToChange != 0 {
		t.Errorf("expected 0 to change, got %d", s.ToChange)
	}
	if s.ToDestroy != 0 {
		t.Errorf("expected 0 to destroy, got %d", s.ToDestroy)
	}
	if len(s.Creates) != 3 {
		t.Errorf("expected 3 create resources, got %d", len(s.Creates))
	}
	if s.Creates[0].Address != "module.s3_bucket.aws_s3_bucket.default[0]" {
		t.Errorf("unexpected first create address: %s", s.Creates[0].Address)
	}
	if s.RawOutput == "" {
		t.Error("expected RawOutput to be populated")
	}
}

func TestParsePlanReplace(t *testing.T) {
	s, err := internal.Parse(planReplaceOutput, internal.PhasePlan, "plat-ue2-sandbox", false)
	if err != nil {
		t.Fatal(err)
	}

	if s.ToAdd != 1 {
		t.Errorf("expected 1 to add, got %d", s.ToAdd)
	}
	if s.ToChange != 1 {
		t.Errorf("expected 1 to change, got %d", s.ToChange)
	}
	if s.ToDestroy != 2 {
		t.Errorf("expected 2 to destroy, got %d", s.ToDestroy)
	}
	if len(s.Replaces) != 1 {
		t.Errorf("expected 1 replace resource, got %d", len(s.Replaces))
	}
	if len(s.Updates) != 1 {
		t.Errorf("expected 1 update resource, got %d", len(s.Updates))
	}
	if len(s.Destroys) != 1 {
		t.Errorf("expected 1 destroy resource, got %d", len(s.Destroys))
	}
	if len(s.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(s.Warnings))
	}
}

func TestParseApplySuccess(t *testing.T) {
	s, err := internal.Parse(applySuccessOutput, internal.PhaseApply, "prod", false)
	if err != nil {
		t.Fatal(err)
	}

	if !s.ApplySucceeded {
		t.Error("expected apply to succeed")
	}
	if s.ToAdd != 2 {
		t.Errorf("expected 2 added, got %d", s.ToAdd)
	}
	if len(s.Creates) != 2 {
		t.Errorf("expected 2 created resources, got %d", len(s.Creates))
	}
	if len(s.Failures) != 0 {
		t.Errorf("expected 0 failures, got %d", len(s.Failures))
	}
}

func TestParseApplyMixed(t *testing.T) {
	s, err := internal.Parse(applyMixedOutput, internal.PhaseApply, "prod", false)
	if err != nil {
		t.Fatal(err)
	}

	if s.ApplySucceeded {
		t.Error("expected apply to not fully succeed")
	}
	if len(s.Creates) != 1 {
		t.Errorf("expected 1 created resource, got %d", len(s.Creates))
	}
	if len(s.Failures) != 1 {
		t.Errorf("expected 1 failure, got %d", len(s.Failures))
	}
	if len(s.Failures) > 0 && s.Failures[0].Address != "module.rds.aws_db_instance.main" {
		t.Errorf("expected failure for module.rds.aws_db_instance.main, got %s", s.Failures[0].Address)
	}
	if len(s.Failures) > 0 && s.Failures[0].Error == "" {
		t.Error("expected failure to have error message")
	}
}

func TestParseApplyFail(t *testing.T) {
	s, err := internal.Parse(applyFailOutput, internal.PhaseApply, "prod", false)
	if err != nil {
		t.Fatal(err)
	}

	if s.ApplySucceeded {
		t.Error("expected apply to fail")
	}
	if len(s.Errors) == 0 {
		t.Error("expected at least one error")
	}
	if len(s.Failures) != 1 {
		t.Errorf("expected 1 failure, got %d", len(s.Failures))
	}
	if len(s.Failures) > 0 && s.Failures[0].Address != "module.rds.aws_db_instance.main" {
		t.Errorf("expected failure address module.rds.aws_db_instance.main, got %s", s.Failures[0].Address)
	}
}

func TestParseApplyDestroy(t *testing.T) {
	s, err := internal.Parse(applyDestroyOutput, internal.PhaseApply, "staging", false)
	if err != nil {
		t.Fatal(err)
	}

	if !s.ApplySucceeded {
		t.Error("expected apply to succeed")
	}
	if len(s.Destroys) != 1 {
		t.Errorf("expected 1 destroy, got %d", len(s.Destroys))
	}
	if len(s.Updates) != 1 {
		t.Errorf("expected 1 update, got %d", len(s.Updates))
	}
	if s.ToDestroy != 1 {
		t.Errorf("expected ToDestroy=1, got %d", s.ToDestroy)
	}
	if s.ToChange != 1 {
		t.Errorf("expected ToChange=1, got %d", s.ToChange)
	}
}

func TestParseNoChanges(t *testing.T) {
	s, err := internal.Parse(noChangesOutput, internal.PhasePlan, "dev", false)
	if err != nil {
		t.Fatal(err)
	}

	if s.ToAdd != 0 || s.ToChange != 0 || s.ToDestroy != 0 {
		t.Error("expected all zeros for no changes")
	}
}

func TestParseDrift(t *testing.T) {
	s, err := internal.Parse(driftOutput, internal.PhasePlan, "staging", false)
	if err != nil {
		t.Fatal(err)
	}

	if !s.DriftDetected {
		t.Error("expected drift to be detected")
	}
}

func TestParsePlanCreateWithANSI(t *testing.T) {
	s, err := internal.Parse(planCreateWithANSIOutput, internal.PhasePlan, "oci_maklab_base0", false)
	if err != nil {
		t.Fatal(err)
	}

	// The plan summary line says 15 to add, but test data only has 3 resources defined
	// This test verifies that ANSI codes are properly stripped and parsing works
	if s.ToAdd != 15 {
		t.Errorf("expected 15 to add (from plan summary), got %d", s.ToAdd)
	}
	if s.ToChange != 0 {
		t.Errorf("expected 0 to change, got %d", s.ToChange)
	}
	if s.ToDestroy != 0 {
		t.Errorf("expected 0 to destroy, got %d", s.ToDestroy)
	}
	// Test data only has 3 creates defined, not 15
	if len(s.Creates) != 3 {
		t.Errorf("expected 3 create resources in test data, got %d", len(s.Creates))
	}
	if len(s.Reads) != 2 {
		t.Errorf("expected 2 read resources, got %d", len(s.Reads))
	}
	// Verify ANSI codes are stripped from addresses
	if s.Creates[0].Address != "doppler_secret.tailscale_auth_key" {
		t.Errorf("unexpected first create address: %s (ANSI codes not stripped?)", s.Creates[0].Address)
	}
	if s.Reads[0].Address != "module.compute_instance.data.oci_core_private_ips.private_ips[0]" {
		t.Errorf("unexpected first read address: %s (ANSI codes not stripped?)", s.Reads[0].Address)
	}
}

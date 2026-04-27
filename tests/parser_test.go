package tests

import (
	"strings"
	"testing"

	"github.com/jomakori/TF_summarize/internal"
	"github.com/jomakori/TF_summarize/internal/parser"
)

// assertCount is a helper to check resource counts
func assertCount(t *testing.T, actual, expected int, name string) {
	t.Helper()
	if actual != expected {
		t.Errorf("expected %d %s, got %d", expected, name, actual)
	}
}

// assertAddress is a helper to check resource address
func assertAddress(t *testing.T, actual, expected, name string) {
	t.Helper()
	if actual != expected {
		t.Errorf("expected %s address '%s', got '%s'", name, expected, actual)
	}
}

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
	s, err := parser.Parse(planCreateOutput, internal.PhasePlan, "plat-ue2-sandbox", false)
	if err != nil {
		t.Fatal(err)
	}

	assertCount(t, s.ToAdd, 3, "to add")
	assertCount(t, s.ToChange, 0, "to change")
	assertCount(t, s.ToDestroy, 0, "to destroy")
	assertCount(t, len(s.Creates), 3, "create resources")

	if len(s.Creates) > 0 {
		assertAddress(t, s.Creates[0].Address, "module.s3_bucket.aws_s3_bucket.default[0]", "first create")
	}
	if s.RawOutput == "" {
		t.Error("expected RawOutput to be populated")
	}
}

func TestParsePlanReplace(t *testing.T) {
	s, err := parser.Parse(planReplaceOutput, internal.PhasePlan, "plat-ue2-sandbox", false)
	if err != nil {
		t.Fatal(err)
	}

	assertCount(t, s.ToAdd, 1, "to add")
	assertCount(t, s.ToChange, 1, "to change")
	assertCount(t, s.ToDestroy, 2, "to destroy")
	assertCount(t, len(s.Replaces), 1, "replace resources")
	assertCount(t, len(s.Updates), 1, "update resources")
	assertCount(t, len(s.Destroys), 1, "destroy resources")
	assertCount(t, len(s.Warnings), 1, "warnings")
}

func TestParseApplySuccess(t *testing.T) {
	s, err := parser.Parse(applySuccessOutput, internal.PhaseApply, "prod", false)
	if err != nil {
		t.Fatal(err)
	}

	if !s.ApplySucceeded {
		t.Error("expected apply to succeed")
	}
	assertCount(t, s.ToAdd, 2, "added")
	assertCount(t, len(s.Creates), 2, "created resources")
	assertCount(t, len(s.Failures), 0, "failures")
}

func TestParseApplyMixed(t *testing.T) {
	s, err := parser.Parse(applyMixedOutput, internal.PhaseApply, "prod", false)
	if err != nil {
		t.Fatal(err)
	}

	if s.ApplySucceeded {
		t.Error("expected apply to not fully succeed")
	}
	assertCount(t, len(s.Creates), 1, "created resources")
	assertCount(t, len(s.Failures), 1, "failures")

	if len(s.Failures) > 0 {
		assertAddress(t, s.Failures[0].Address, "module.rds.aws_db_instance.main", "failure")
		if s.Failures[0].Error == "" {
			t.Error("expected failure to have error message")
		}
	}
}

func TestParseApplyFail(t *testing.T) {
	s, err := parser.Parse(applyFailOutput, internal.PhaseApply, "prod", false)
	if err != nil {
		t.Fatal(err)
	}

	if s.ApplySucceeded {
		t.Error("expected apply to fail")
	}
	if len(s.Errors) == 0 {
		t.Error("expected at least one error")
	}
	assertCount(t, len(s.Failures), 1, "failures")

	if len(s.Failures) > 0 {
		assertAddress(t, s.Failures[0].Address, "module.rds.aws_db_instance.main", "failure")
	}
}

func TestParseApplyDestroy(t *testing.T) {
	s, err := parser.Parse(applyDestroyOutput, internal.PhaseApply, "staging", false)
	if err != nil {
		t.Fatal(err)
	}

	if !s.ApplySucceeded {
		t.Error("expected apply to succeed")
	}
	assertCount(t, len(s.Destroys), 1, "destroys")
	assertCount(t, len(s.Updates), 1, "updates")
	assertCount(t, s.ToDestroy, 1, "to destroy")
	assertCount(t, s.ToChange, 1, "to change")
}

func TestParseNoChanges(t *testing.T) {
	s, err := parser.Parse(noChangesOutput, internal.PhasePlan, "dev", false)
	if err != nil {
		t.Fatal(err)
	}

	assertCount(t, s.ToAdd, 0, "to add")
	assertCount(t, s.ToChange, 0, "to change")
	assertCount(t, s.ToDestroy, 0, "to destroy")
}

func TestParseDrift(t *testing.T) {
	s, err := parser.Parse(driftOutput, internal.PhasePlan, "staging", false)
	if err != nil {
		t.Fatal(err)
	}

	if !s.DriftDetected {
		t.Error("expected drift to be detected")
	}
}

func TestParseSymbolsFormat(t *testing.T) {
	// Test that symbol-prefixed lines are correctly parsed as resources
	// and that attribute values (like CIDR blocks, IPs, etc.) are not mistaken for resources
	input := `
  # module.vcn.oci_core_vcn.vcn will be created
  + resource "oci_core_vcn" "vcn" ***
      + cidr_blocks                      = [
          + "10.0.0.0/16",
        ]
      + compartment_id                   = (sensitive value)
      + display_name                     = "maklab-base0-vcn"
    ***

  # module.compute.aws_instance.web will be destroyed
  - resource "aws_instance" "web" {
      - private_ip = "192.168.1.10"
    }

  # module.storage.aws_s3_bucket.data will be updated
  ~ resource "aws_s3_bucket" "data" {
      ~ versioning {
          ~ enabled = true
        }
    }

Plan: 1 to add, 1 to change, 1 to destroy.
`

	s, err := parser.Parse(input, internal.PhasePlan, "test", false)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify correct counts and resources
	tests := []struct {
		name     string
		actual   []internal.ResourceChange
		expected int
		address  string
	}{
		{"creates", s.Creates, 1, "module.vcn.oci_core_vcn.vcn"},
		{"destroys", s.Destroys, 1, "module.compute.aws_instance.web"},
		{"updates", s.Updates, 1, "module.storage.aws_s3_bucket.data"},
	}

	for _, tc := range tests {
		if len(tc.actual) != tc.expected {
			t.Errorf("expected %d %s, got %d", tc.expected, tc.name, len(tc.actual))
		}
		if len(tc.actual) > 0 && tc.actual[0].Address != tc.address {
			t.Errorf("expected %s address '%s', got '%s'", tc.name, tc.address, tc.actual[0].Address)
		}
	}

	// Verify attribute values were not parsed as resources
	allResources := append(append(s.Creates, s.Destroys...), s.Updates...)
	for _, r := range allResources {
		if strings.Contains(r.Address, "10.0.0.0") || strings.Contains(r.Address, "192.168.1.10") {
			t.Errorf("attribute value was incorrectly parsed as resource: %s", r.Address)
		}
	}
}

func TestParsePlanCreateWithANSI(t *testing.T) {
	s, err := parser.Parse(planCreateWithANSIOutput, internal.PhasePlan, "oci_maklab_base0", false)
	if err != nil {
		t.Fatal(err)
	}

	// The plan summary line says 15 to add, but test data only has 3 resources defined
	// This test verifies that ANSI codes are properly stripped and parsing works
	assertCount(t, s.ToAdd, 15, "to add (from plan summary)")
	assertCount(t, s.ToChange, 0, "to change")
	assertCount(t, s.ToDestroy, 0, "to destroy")
	assertCount(t, len(s.Creates), 3, "create resources in test data")
	assertCount(t, len(s.Reads), 2, "read resources")

	// Verify ANSI codes are stripped from addresses
	if len(s.Creates) > 0 {
		assertAddress(t, s.Creates[0].Address, "doppler_secret.tailscale_auth_key", "first create")
	}
	if len(s.Reads) > 0 {
		assertAddress(t, s.Reads[0].Address, "module.compute_instance.data.oci_core_private_ips.private_ips[0]", "first read")
	}
}

// Test for terraform CLI errors with box format (e.g., "Too many command line arguments")
const cliErrorOutput = `
╷
│ Error: Too many command line arguments
│
│ Expected at most one positional argument.
╵

For more help on using this command, run:
  terraform apply -help
`

func TestParseCLIError(t *testing.T) {
	s, err := parser.Parse(cliErrorOutput, internal.PhaseApply, "test", false)
	if err != nil {
		t.Fatal(err)
	}

	if len(s.Errors) == 0 {
		t.Error("expected CLI error to be detected")
	}

	if len(s.Errors) > 0 && !strings.Contains(s.Errors[0], "Too many command line arguments") {
		t.Errorf("expected error message to contain 'Too many command line arguments', got: %s", s.Errors[0])
	}

	// Apply should not be marked as succeeded when there's a CLI error
	if s.ApplySucceeded {
		t.Error("expected ApplySucceeded to be false when CLI error detected")
	}
}

// Test for terraform CLI errors without box format
const cliErrorSimpleOutput = `
Error: Too many command line arguments

Expected at most one positional argument.
`

func TestParseCLIErrorSimple(t *testing.T) {
	s, err := parser.Parse(cliErrorSimpleOutput, internal.PhaseApply, "test", false)
	if err != nil {
		t.Fatal(err)
	}

	if len(s.Errors) == 0 {
		t.Error("expected CLI error to be detected")
	}

	if s.ApplySucceeded {
		t.Error("expected ApplySucceeded to be false when CLI error detected")
	}
}

// Test for terraform apply with invalid flag error
const invalidFlagErrorOutput = `
╷
│ Error: Failed to parse command-line flags
│
│ flag provided but not defined: -invalid-flag
╵
`

func TestParseInvalidFlagError(t *testing.T) {
	s, err := parser.Parse(invalidFlagErrorOutput, internal.PhaseApply, "test", false)
	if err != nil {
		t.Fatal(err)
	}

	if len(s.Errors) == 0 {
		t.Error("expected invalid flag error to be detected")
	}

	if s.ApplySucceeded {
		t.Error("expected ApplySucceeded to be false when invalid flag error detected")
	}
}

// Test for GitHub Actions error annotation format
const ghaErrorAnnotationOutput = `
::error::Terraform exited with code 1.
`

func TestParseGHAErrorAnnotation(t *testing.T) {
	s, err := parser.Parse(ghaErrorAnnotationOutput, internal.PhaseApply, "test", false)
	if err != nil {
		t.Fatal(err)
	}

	if len(s.Errors) == 0 {
		t.Error("expected GHA error annotation to be detected")
	}

	if len(s.Errors) > 0 && !strings.Contains(s.Errors[0], "Terraform exited with code 1") {
		t.Errorf("expected error message to contain 'Terraform exited with code 1', got: %s", s.Errors[0])
	}

	if s.ApplySucceeded {
		t.Error("expected ApplySucceeded to be false when GHA error annotation detected")
	}
}

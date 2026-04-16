package tests

import (
	"strings"
	"testing"

	"github.com/jomakori/TF_summarize/internal"
)

func TestParseCIDRBlockNotParsedAsResource(t *testing.T) {
	// Test with the actual terraform plan output that shows the issue
	// The plan says "15 to add" but the CIDR block line "+ "10.0.0.0/16"," 
	// should NOT be counted as a resource
	input := `
Terraform used the selected providers to generate the following execution plan.

  # module.vcn.oci_core_vcn.vcn will be created
  + resource "oci_core_vcn" "vcn" {
      + cidr_blocks                      = [
          + "10.0.0.0/16",
        ]
      + display_name                     = "maklab-base0-vcn"
    }

  # module.vcn.oci_core_internet_gateway.ig[0] will be created
  + resource "oci_core_internet_gateway" "ig" {
      + display_name                     = "maklab-base0-igw"
    }

Plan: 2 to add, 0 to change, 0 to destroy.
`

	s, err := internal.Parse(input, internal.PhasePlan, "test", false)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have 2 creates (VCN and IGW), not 3 (VCN + CIDR + IGW)
	if len(s.Creates) != 2 {
		t.Errorf("expected 2 creates, got %d", len(s.Creates))
		for i, c := range s.Creates {
			t.Logf("  Create[%d]: %s", i, c.Address)
		}
	}

	// Verify CIDR block was not parsed as a resource
	for _, c := range s.Creates {
		if strings.Contains(c.Address, "10.0.0.0") || strings.Contains(c.Address, "\"") {
			t.Errorf("CIDR block or quoted string was incorrectly parsed as resource: %s", c.Address)
		}
	}

	// Verify the plan summary count matches the actual creates
	if s.ToAdd != len(s.Creates) {
		t.Errorf("plan summary says %d to add, but found %d creates", s.ToAdd, len(s.Creates))
	}
}

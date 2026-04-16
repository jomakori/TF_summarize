package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jomakori/TF_summarize/internal"
	"github.com/jomakori/TF_summarize/internal/providers"
)

// Version is set at build time using ldflags
// Example: go build -ldflags "-X main.Version=1.0.0"
var Version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	// --- Parse command-line flags ---
	versionFlag := flag.Bool("version", false, "display version and exit")
	helpFlag := flag.Bool("help", false, "display help and exit")
	hFlag := flag.Bool("h", false, "display help and exit (shorthand)")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("tf-summarize version %s\n", Version)
		return nil
	}

	if *helpFlag || *hFlag {
		printHelp()
		return nil
	}

	// --- Read configuration from env vars ---

	workspace := os.Getenv("TF_WORKSPACE")
	if workspace == "" {
		workspace = os.Getenv("WORKSPACE") // fallback
	}
	if workspace == "" {
		workspace = "default"
	}

	// Phase: "plan" or "apply"
	phaseStr := strings.ToLower(os.Getenv("TF_PHASE"))
	if phaseStr == "" {
		phaseStr = "plan"
	}
	var phase internal.Phase
	switch phaseStr {
	case "apply":
		phase = internal.PhaseApply
	default:
		phase = internal.PhasePlan
	}

	// Check if this is a destroy plan
	isDestroyPlan := false
	if destroyEnv := os.Getenv("DESTROY"); destroyEnv != "" {
		isDestroyPlan = strings.ToLower(destroyEnv) == "true" || destroyEnv == "1"
	}

	// Output target: "gha", "pr", "stdout" (default), or comma-separated combo
	targetStr := strings.ToLower(os.Getenv("TF_OUTPUT"))
	if targetStr == "" {
		targetStr = "stdout"
	}

	// --- Read terraform output ---
	inputFile := os.Getenv("TF_PLAN_FILE")
	jsonPlanFile := os.Getenv("TF_PLAN_JSON")
	var input string
	var summary *internal.Summary
	var err error

	// Try JSON plan first if available
	if jsonPlanFile != "" {
		data, err := os.ReadFile(jsonPlanFile)
		if err != nil {
			return fmt.Errorf("reading JSON plan file %s: %w", jsonPlanFile, err)
		}
		summary, err = internal.ParsePlanJSON(data, workspace, isDestroyPlan)
		if err != nil {
			return fmt.Errorf("parsing JSON plan: %w", err)
		}
		summary.Phase = phase
	} else {
		// Fall back to text parsing
		if inputFile != "" {
			data, err := os.ReadFile(inputFile)
			if err != nil {
				return fmt.Errorf("reading plan file %s: %w", inputFile, err)
			}
			input = string(data)
		} else {
			// Read from stdin
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("reading stdin: %w", err)
			}
			input = string(data)
		}

		if strings.TrimSpace(input) == "" {
			return fmt.Errorf("no input provided — set TF_PLAN_FILE, TF_PLAN_JSON, or pipe terraform output to stdin")
		}

		// --- Parse & render ---
		summary, err = internal.Parse(input, phase, workspace, isDestroyPlan)
		if err != nil {
			return fmt.Errorf("parsing terraform output: %w", err)
		}
	}

	markdown := internal.Render(summary)

	// --- Output using provider pattern ---
	targets := strings.Split(targetStr, ",")
	for _, t := range targets {
		t = strings.TrimSpace(t)
		var provider internal.OutputProvider

		switch t {
		case "gha":
			provider = providers.NewGitHubProvider()
		case "pr":
			provider = providers.NewGitHubProvider()
		case "stdout":
			provider = providers.NewStdoutProvider()
		default:
			return fmt.Errorf("unknown output target: %q (use gha, pr, or stdout)", t)
		}

		if err := provider.WriteSummary(summary, markdown); err != nil {
			return fmt.Errorf("writing output via %s provider: %w", provider.Name(), err)
		}
	}

	// Set exit code based on changes (useful for CI)
	if os.Getenv("TF_EXIT_ON_CHANGES") == "true" {
		if summary.ToAdd > 0 || summary.ToChange > 0 || summary.ToDestroy > 0 {
			os.Exit(2) // signal "changes detected" without being a real error
		}
	}

	return nil
}

func printHelp() {
	fmt.Printf(`tf-summarize v%s

A CLI tool that parses Terraform plan and apply output and produces
beautified Markdown summaries suitable for GitHub Actions or PR comments.

Supports both text output parsing and structured JSON plan parsing for
accurate resource change detection.

USAGE:
  tf-summarize [flags]

FLAGS:
  -version    Display version and exit
  -help, -h   Display this help message and exit

ENVIRONMENT VARIABLES:
  TF_PLAN_FILE        Path to file containing terraform plan/apply text output (default: stdin)
  TF_PLAN_JSON        Path to terraform plan JSON file (from 'terraform show -json <planfile>')
                      When set, uses structured JSON parsing for accurate counts
  TF_WORKSPACE        Workspace name shown in header (default: "default")
  TF_PHASE            "plan" or "apply" - controls header messaging (default: "plan")
  TF_OUTPUT           Output target(s): "stdout", "gha", "pr" (comma-separated, default: "stdout")
  GITHUB_TOKEN        GitHub token for posting PR comments (required for "pr" output)
  GITHUB_REPOSITORY   Repository in "owner/repo" format (set automatically in GitHub Actions)
  PR_NUMBER           Pull request number to comment on (required for "pr" output)
  GITHUB_API_URL      GitHub API base URL (default: "https://api.github.com")
  TF_EXIT_ON_CHANGES  Exit with code 2 when changes detected (default: "false")
  DESTROY             Set to "true" or "1" for destroy plans (changes phase badge to red)

EXAMPLES:
  # Pipe terraform plan output (text parsing)
  terraform plan -no-color | tf-summarize

  # Use a saved plan file (text parsing)
  TF_PLAN_FILE=plan.txt tf-summarize

  # Use JSON plan for accurate parsing (recommended)
  terraform show -json plan.tfplan > plan.json
  TF_PLAN_JSON=plan.json tf-summarize

  # Apply phase with GitHub Actions output
  terraform apply -no-color -auto-approve 2>&1 | TF_PHASE=apply TF_OUTPUT=gha tf-summarize

  # Destroy plan with caution badge
  terraform plan -destroy -no-color | DESTROY=true tf-summarize

For more information, visit: https://github.com/jomakori/TF_summarize
`, Version)
}

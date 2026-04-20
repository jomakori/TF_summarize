package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jomakori/TF_summarize/internal"
	"github.com/jomakori/TF_summarize/internal/parser"
	"github.com/jomakori/TF_summarize/internal/providers"
	"github.com/jomakori/TF_summarize/internal/render"
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
	workspace := internal.GetEnvWithFallback("default", "TF_WORKSPACE", "WORKSPACE")
	phaseStr := strings.ToLower(internal.GetEnv("TF_PHASE", "plan"))
	isDestroyPlan := internal.GetEnvBool("DESTROY")
	targetStr := strings.ToLower(internal.GetEnv("TF_OUTPUT", "stdout"))
	inputFile := internal.GetEnv("TF_PLAN_FILE", "")
	jsonPlanFile := internal.GetEnv("TF_PLAN_JSON", "")

	var phase internal.Phase
	switch phaseStr {
	case "apply":
		phase = internal.PhaseApply
	default:
		phase = internal.PhasePlan
	}
	var input string
	var summary *internal.Summary
	var err error

	// For apply phase, always read from stdin to capture actual apply output
	// JSON plan is only useful for plan phase (pre-apply analysis)
	if phase == internal.PhaseApply {
		// Read from stdin for apply output
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		input = string(data)

		if strings.TrimSpace(input) == "" {
			return fmt.Errorf("no terraform apply output received — terraform may have failed before producing output")
		}

		summary, err = parser.Parse(input, phase, workspace, isDestroyPlan)
		if err != nil {
			return fmt.Errorf("parsing terraform output: %w", err)
		}
	} else if jsonPlanFile != "" {
		// For plan phase, try JSON plan first if available
		data, err := os.ReadFile(jsonPlanFile)
		if err != nil {
			return fmt.Errorf("reading JSON plan file %s: %w", jsonPlanFile, err)
		}
		summary, err = parser.ParsePlanJSON(data, workspace, isDestroyPlan)
		if err != nil {
			return fmt.Errorf("parsing JSON plan: %w", err)
		}
		summary.Phase = phase
	} else {
		// Fall back to text parsing for plan phase
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
		summary, err = parser.Parse(input, phase, workspace, isDestroyPlan)
		if err != nil {
			return fmt.Errorf("parsing terraform output: %w", err)
		}
	}

	// --- Output using provider pattern ---
	// Each target does exactly what it says - no implicit behavior
	// Render separately for each provider to apply provider-specific formatting
	targets := strings.Split(targetStr, ",")

	for _, t := range targets {
		t = strings.TrimSpace(t)
		var provider internal.OutputProvider
		var targetProvider internal.OutputTarget

		switch t {
		case "gha":
			// Writes to GitHub Actions step summary only
			provider = providers.NewGitHubProvider()
			targetProvider = internal.TargetGHASummary
		case "pr":
			// Writes to PR comment only (does NOT write to GHA step summary)
			provider = providers.NewGitHubPRProvider()
			targetProvider = internal.TargetPR
		case "stdout":
			provider = providers.NewStdoutProvider()
			targetProvider = internal.TargetStdout
		default:
			return fmt.Errorf("unknown output target: %q (use gha, pr, or stdout)", t)
		}

		// Set the target provider for rendering (affects error formatting)
		summary.TargetProvider = targetProvider
		markdown := render.Render(summary)

		if err := provider.WriteSummary(summary, markdown); err != nil {
			return fmt.Errorf("writing output via %s provider: %w", provider.Name(), err)
		}
	}

	// Exit with error code if terraform errors were detected
	if len(summary.Errors) > 0 || len(summary.Failures) > 0 {
		fmt.Fprintf(os.Stderr, "⚠ Terraform errors detected (%d errors, %d failures) - exiting with code 1\n", len(summary.Errors), len(summary.Failures))
		os.Exit(1) // signal terraform error
	}

	// Set exit code based on changes (useful for CI)
	if internal.GetEnvBool("TF_EXIT_ON_CHANGES") {
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
  PR_NUMBER           Pull request number to comment on (optional - auto-detected from branch)
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

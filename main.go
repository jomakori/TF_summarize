package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jomakori/TF_summarize/internal"
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

	// Output target: "gha", "pr", "stdout" (default), or comma-separated combo
	targetStr := strings.ToLower(os.Getenv("TF_OUTPUT"))
	if targetStr == "" {
		targetStr = "stdout"
	}

	// --- Read terraform output ---
	inputFile := os.Getenv("TF_PLAN_FILE")
	var input string

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
		return fmt.Errorf("no input provided — set TF_PLAN_FILE or pipe terraform output to stdin")
	}

	// --- Parse & render ---
	summary, err := internal.Parse(input, phase, workspace)
	if err != nil {
		return fmt.Errorf("parsing terraform output: %w", err)
	}

	markdown := internal.Render(summary)

	// --- Output ---
	targets := strings.Split(targetStr, ",")
	for _, t := range targets {
		switch strings.TrimSpace(t) {
		case "gha":
			if err := internal.WriteGHASummary(markdown); err != nil {
				return fmt.Errorf("writing GHA summary: %w", err)
			}
			fmt.Fprintln(os.Stderr, "✓ Written to GitHub Actions step summary")
		case "pr":
			if err := internal.WritePRComment(markdown); err != nil {
				return fmt.Errorf("writing PR comment: %w", err)
			}
			fmt.Fprintln(os.Stderr, "✓ Posted/updated PR comment")
		case "stdout":
			internal.WriteStdout(markdown)
		default:
			return fmt.Errorf("unknown output target: %q (use gha, pr, or stdout)", t)
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

USAGE:
  tf-summarize [flags]

FLAGS:
  -version    Display version and exit
  -help, -h   Display this help message and exit

ENVIRONMENT VARIABLES:
  TF_PLAN_FILE        Path to file containing terraform plan/apply output (default: stdin)
  TF_WORKSPACE        Workspace name shown in header (default: "default")
  TF_PHASE            "plan" or "apply" - controls header messaging (default: "plan")
  TF_OUTPUT           Output target(s): "stdout", "gha", "pr" (comma-separated, default: "stdout")
  GITHUB_TOKEN        GitHub token for posting PR comments (required for "pr" output)
  GITHUB_REPOSITORY   Repository in "owner/repo" format (set automatically in GitHub Actions)
  PR_NUMBER           Pull request number to comment on (required for "pr" output)
  GITHUB_API_URL      GitHub API base URL (default: "https://api.github.com")
  TF_EXIT_ON_CHANGES  Exit with code 2 when changes detected (default: "false")

EXAMPLES:
  # Pipe terraform plan output
  terraform plan -no-color | tf-summarize

  # Use a saved plan file
  TF_PLAN_FILE=plan.txt tf-summarize

  # Apply phase with GitHub Actions output
  terraform apply -no-color -auto-approve 2>&1 | TF_PHASE=apply TF_OUTPUT=gha tf-summarize

For more information, visit: https://github.com/jomakori/TF_summarize
`, Version)
}

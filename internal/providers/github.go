package providers

import (
	"fmt"
	"os"
	"strconv"

	"github.com/jomakori/TF_summarize/internal"
)

// GitHubProvider writes terraform summaries to GitHub Actions and PR comments.
type GitHubProvider struct {
	*BaseProvider
	summaryFile string
	prNumber    int
	token       string
	repo        string
	apiURL      string
	requirePR   bool // If true, fail when no PR is found
}

// NewGitHubProvider creates a new GitHub provider for GHA step summary.
func NewGitHubProvider() *GitHubProvider {
	return &GitHubProvider{
		BaseProvider: NewBaseProvider("github"),
		summaryFile:  os.Getenv("GITHUB_STEP_SUMMARY"),
		token:        os.Getenv("GITHUB_TOKEN"),
		repo:         os.Getenv("GITHUB_REPOSITORY"),
		apiURL:       getEnvOrDefault("GITHUB_API_URL", "https://api.github.com"),
		requirePR:    false,
	}
}

// NewGitHubPRProvider creates a new GitHub provider specifically for PR comments.
// It will attempt to find the PR number dynamically if not provided.
// Note: This provider does NOT write to GHA step summary - use "gha" target for that.
func NewGitHubPRProvider() *GitHubProvider {
	p := &GitHubProvider{
		BaseProvider: NewBaseProvider("github-pr"),
		summaryFile:  "", // PR provider does NOT write to step summary (gha provider does that)
		token:        os.Getenv("GITHUB_TOKEN"),
		repo:         os.Getenv("GITHUB_REPOSITORY"),
		apiURL:       getEnvOrDefault("GITHUB_API_URL", "https://api.github.com"),
		requirePR:    true, // PR provider requires a PR
	}

	// Try to get PR number from environment first
	if prStr := os.Getenv("PR_NUMBER"); prStr != "" {
		if num, err := strconv.Atoi(prStr); err == nil {
			p.prNumber = num
		}
	}

	return p
}

// ResolvePRNumber attempts to find the PR number if not already set.
// For workflow_dispatch or push events, it looks up the PR by branch name.
func (p *GitHubProvider) ResolvePRNumber() error {
	if p.prNumber > 0 {
		return nil // Already have a PR number
	}

	// Check environment variable first
	if prStr := os.Getenv("PR_NUMBER"); prStr != "" {
		if num, err := strconv.Atoi(prStr); err == nil {
			p.prNumber = num
			return nil
		}
	}

	// Try to find PR by branch name
	branch := internal.GetCurrentBranch()
	if branch == "" {
		if p.requirePR {
			return fmt.Errorf("cannot determine current branch — set GITHUB_HEAD_REF, GITHUB_REF_NAME, or GITHUB_REF")
		}
		return nil
	}

	// Skip PR lookup for main/master branches (they typically don't have open PRs)
	if branch == "main" || branch == "master" {
		if p.requirePR {
			return fmt.Errorf("no PR associated with branch '%s' — PR comments require a feature branch with an open PR", branch)
		}
		return nil
	}

	prNum, err := internal.FindPRForBranch(branch)
	if err != nil {
		if p.requirePR {
			return fmt.Errorf("failed to find PR for branch '%s': %w", branch, err)
		}
		fmt.Fprintf(os.Stderr, "⚠ Warning: could not find PR for branch '%s': %v\n", branch, err)
		return nil
	}

	if prNum == 0 {
		if p.requirePR {
			return fmt.Errorf("no open PR found for branch '%s' — create a PR first or remove 'pr' from TF_OUTPUT", branch)
		}
		fmt.Fprintf(os.Stderr, "⚠ Warning: no open PR found for branch '%s'\n", branch)
		return nil
	}

	p.prNumber = prNum
	fmt.Fprintf(os.Stderr, "✓ Found PR #%d for branch '%s'\n", prNum, branch)
	return nil
}

// WriteSummary writes the markdown summary to GitHub Actions step summary and/or PR comment.
func (p *GitHubProvider) WriteSummary(summary *internal.Summary, markdown string) error {
	// Write to GitHub Actions step summary
	if p.summaryFile != "" {
		if err := internal.WriteGHASummary(markdown); err != nil {
			return fmt.Errorf("writing GHA summary: %w", err)
		}
		fmt.Fprintln(os.Stderr, "✓ Written to GitHub Actions step summary")
	}

	// For PR provider, resolve PR number dynamically if needed
	if p.requirePR {
		if err := p.ResolvePRNumber(); err != nil {
			return err
		}

		if p.prNumber > 0 && p.token != "" && p.repo != "" {
			// Set PR_NUMBER env var for WritePRComment to use
			os.Setenv("PR_NUMBER", strconv.Itoa(p.prNumber))
			if err := internal.WritePRComment(markdown); err != nil {
				return fmt.Errorf("writing PR comment: %w", err)
			}
			fmt.Fprintln(os.Stderr, "✓ Posted/updated PR comment")
		}
	} else {
		// Legacy behavior: only write if PR_NUMBER is explicitly set
		prNumber := os.Getenv("PR_NUMBER")
		if prNumber != "" && p.token != "" && p.repo != "" {
			if err := internal.WritePRComment(markdown); err != nil {
				return fmt.Errorf("writing PR comment: %w", err)
			}
			fmt.Fprintln(os.Stderr, "✓ Posted/updated PR comment")
		}
	}

	return nil
}

// WriteOutputs is a no-op for GitHub (outputs are included in summary).
func (p *GitHubProvider) WriteOutputs(summary *internal.Summary, markdown string) error {
	return nil
}

// getEnvOrDefault returns an environment variable or a default value.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

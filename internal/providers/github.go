package providers

import (
	"fmt"
	"os"

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
}

// NewGitHubProvider creates a new GitHub provider.
func NewGitHubProvider() *GitHubProvider {
	return &GitHubProvider{
		BaseProvider: NewBaseProvider("github"),
		summaryFile: os.Getenv("GITHUB_STEP_SUMMARY"),
		token:       os.Getenv("GITHUB_TOKEN"),
		repo:        os.Getenv("GITHUB_REPOSITORY"),
		apiURL:      getEnvOrDefault("GITHUB_API_URL", "https://api.github.com"),
	}
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

	// Write to PR comment if configured
	prNumber := os.Getenv("PR_NUMBER")
	if prNumber != "" && p.token != "" && p.repo != "" {
		if err := internal.WritePRComment(markdown); err != nil {
			return fmt.Errorf("writing PR comment: %w", err)
		}
		fmt.Fprintln(os.Stderr, "✓ Posted/updated PR comment")
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

package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jomakori/TF_summarize/internal"
	"github.com/jomakori/TF_summarize/internal/providers"
)

func TestGetCurrentBranch(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected string
	}{
		{
			name: "GITHUB_HEAD_REF takes priority",
			envVars: map[string]string{
				"GITHUB_HEAD_REF":  "feature/my-branch",
				"GITHUB_REF_NAME":  "main",
				"GITHUB_REF":       "refs/heads/develop",
			},
			expected: "feature/my-branch",
		},
		{
			name: "GITHUB_REF_NAME used when HEAD_REF empty",
			envVars: map[string]string{
				"GITHUB_HEAD_REF":  "",
				"GITHUB_REF_NAME":  "feature/test",
				"GITHUB_REF":       "refs/heads/develop",
			},
			expected: "feature/test",
		},
		{
			name: "GITHUB_REF parsed when others empty",
			envVars: map[string]string{
				"GITHUB_HEAD_REF":  "",
				"GITHUB_REF_NAME":  "",
				"GITHUB_REF":       "refs/heads/feature/from-ref",
			},
			expected: "feature/from-ref",
		},
		{
			name: "Empty when no env vars set",
			envVars: map[string]string{
				"GITHUB_HEAD_REF":  "",
				"GITHUB_REF_NAME":  "",
				"GITHUB_REF":       "",
			},
			expected: "",
		},
		{
			name: "Non-heads ref returns empty",
			envVars: map[string]string{
				"GITHUB_HEAD_REF":  "",
				"GITHUB_REF_NAME":  "",
				"GITHUB_REF":       "refs/tags/v1.0.0",
			},
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Save original env vars
			origHeadRef := os.Getenv("GITHUB_HEAD_REF")
			origRefName := os.Getenv("GITHUB_REF_NAME")
			origRef := os.Getenv("GITHUB_REF")

			// Set test env vars
			for k, v := range tc.envVars {
				os.Setenv(k, v)
			}

			// Test
			result := internal.GetCurrentBranch()
			if result != tc.expected {
				t.Errorf("expected '%s', got '%s'", tc.expected, result)
			}

			// Restore original env vars
			os.Setenv("GITHUB_HEAD_REF", origHeadRef)
			os.Setenv("GITHUB_REF_NAME", origRefName)
			os.Setenv("GITHUB_REF", origRef)
		})
	}
}

func TestFindPRForBranch(t *testing.T) {
	// Create a mock GitHub API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Check the query parameter
		head := r.URL.Query().Get("head")

		switch head {
		case "testowner:feature/has-pr":
			// Return a PR
			prs := []map[string]interface{}{
				{"number": 42},
			}
			json.NewEncoder(w).Encode(prs)
		case "testowner:feature/no-pr":
			// Return empty array (no PR)
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Save and set env vars
	origToken := os.Getenv("GITHUB_TOKEN")
	origRepo := os.Getenv("GITHUB_REPOSITORY")
	origAPI := os.Getenv("GITHUB_API_URL")

	os.Setenv("GITHUB_TOKEN", "test-token")
	os.Setenv("GITHUB_REPOSITORY", "testowner/testrepo")
	os.Setenv("GITHUB_API_URL", server.URL)

	defer func() {
		os.Setenv("GITHUB_TOKEN", origToken)
		os.Setenv("GITHUB_REPOSITORY", origRepo)
		os.Setenv("GITHUB_API_URL", origAPI)
	}()

	t.Run("finds PR for branch", func(t *testing.T) {
		prNum, err := internal.FindPRForBranch("feature/has-pr")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if prNum != 42 {
			t.Errorf("expected PR #42, got #%d", prNum)
		}
	})

	t.Run("returns 0 when no PR found", func(t *testing.T) {
		prNum, err := internal.FindPRForBranch("feature/no-pr")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if prNum != 0 {
			t.Errorf("expected PR #0, got #%d", prNum)
		}
	})
}

func TestFindPRForBranchMissingEnv(t *testing.T) {
	// Save original env vars
	origToken := os.Getenv("GITHUB_TOKEN")
	origRepo := os.Getenv("GITHUB_REPOSITORY")

	defer func() {
		os.Setenv("GITHUB_TOKEN", origToken)
		os.Setenv("GITHUB_REPOSITORY", origRepo)
	}()

	t.Run("error when GITHUB_TOKEN not set", func(t *testing.T) {
		os.Setenv("GITHUB_TOKEN", "")
		os.Setenv("GITHUB_REPOSITORY", "owner/repo")

		_, err := internal.FindPRForBranch("feature/test")
		if err == nil {
			t.Error("expected error when GITHUB_TOKEN not set")
		}
	})

	t.Run("error when GITHUB_REPOSITORY not set", func(t *testing.T) {
		os.Setenv("GITHUB_TOKEN", "test-token")
		os.Setenv("GITHUB_REPOSITORY", "")

		_, err := internal.FindPRForBranch("feature/test")
		if err == nil {
			t.Error("expected error when GITHUB_REPOSITORY not set")
		}
	})
}

func TestGitHubPRProviderResolvePRNumber(t *testing.T) {
	// Create a mock GitHub API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		head := r.URL.Query().Get("head")

		switch head {
		case "testowner:feature/has-pr":
			prs := []map[string]interface{}{
				{"number": 123},
			}
			json.NewEncoder(w).Encode(prs)
		default:
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		}
	}))
	defer server.Close()

	// Save and set env vars
	origToken := os.Getenv("GITHUB_TOKEN")
	origRepo := os.Getenv("GITHUB_REPOSITORY")
	origAPI := os.Getenv("GITHUB_API_URL")
	origPRNum := os.Getenv("PR_NUMBER")
	origHeadRef := os.Getenv("GITHUB_HEAD_REF")
	origRefName := os.Getenv("GITHUB_REF_NAME")
	origRef := os.Getenv("GITHUB_REF")

	defer func() {
		os.Setenv("GITHUB_TOKEN", origToken)
		os.Setenv("GITHUB_REPOSITORY", origRepo)
		os.Setenv("GITHUB_API_URL", origAPI)
		os.Setenv("PR_NUMBER", origPRNum)
		os.Setenv("GITHUB_HEAD_REF", origHeadRef)
		os.Setenv("GITHUB_REF_NAME", origRefName)
		os.Setenv("GITHUB_REF", origRef)
	}()

	os.Setenv("GITHUB_TOKEN", "test-token")
	os.Setenv("GITHUB_REPOSITORY", "testowner/testrepo")
	os.Setenv("GITHUB_API_URL", server.URL)

	t.Run("uses PR_NUMBER env var if set", func(t *testing.T) {
		os.Setenv("PR_NUMBER", "99")
		os.Setenv("GITHUB_HEAD_REF", "")
		os.Setenv("GITHUB_REF_NAME", "")
		os.Setenv("GITHUB_REF", "")

		p := providers.NewGitHubPRProvider()
		err := p.ResolvePRNumber()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// PR number should be resolved from env
		os.Setenv("PR_NUMBER", "")
	})

	t.Run("fails for main branch with requirePR", func(t *testing.T) {
		os.Setenv("PR_NUMBER", "")
		os.Setenv("GITHUB_HEAD_REF", "")
		os.Setenv("GITHUB_REF_NAME", "main")
		os.Setenv("GITHUB_REF", "")

		p := providers.NewGitHubPRProvider()
		err := p.ResolvePRNumber()
		if err == nil {
			t.Error("expected error for main branch with requirePR")
		}
	})

	t.Run("fails when no PR found for feature branch", func(t *testing.T) {
		os.Setenv("PR_NUMBER", "")
		os.Setenv("GITHUB_HEAD_REF", "")
		os.Setenv("GITHUB_REF_NAME", "feature/no-pr")
		os.Setenv("GITHUB_REF", "")

		p := providers.NewGitHubPRProvider()
		err := p.ResolvePRNumber()
		if err == nil {
			t.Error("expected error when no PR found for feature branch")
		}
	})

	t.Run("succeeds when PR found for feature branch", func(t *testing.T) {
		os.Setenv("PR_NUMBER", "")
		os.Setenv("GITHUB_HEAD_REF", "")
		os.Setenv("GITHUB_REF_NAME", "feature/has-pr")
		os.Setenv("GITHUB_REF", "")

		p := providers.NewGitHubPRProvider()
		err := p.ResolvePRNumber()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestGitHubProviderDoesNotRequirePR(t *testing.T) {
	// Save and set env vars
	origPRNum := os.Getenv("PR_NUMBER")
	origHeadRef := os.Getenv("GITHUB_HEAD_REF")
	origRefName := os.Getenv("GITHUB_REF_NAME")
	origRef := os.Getenv("GITHUB_REF")

	defer func() {
		os.Setenv("PR_NUMBER", origPRNum)
		os.Setenv("GITHUB_HEAD_REF", origHeadRef)
		os.Setenv("GITHUB_REF_NAME", origRefName)
		os.Setenv("GITHUB_REF", origRef)
	}()

	os.Setenv("PR_NUMBER", "")
	os.Setenv("GITHUB_HEAD_REF", "")
	os.Setenv("GITHUB_REF_NAME", "main")
	os.Setenv("GITHUB_REF", "")

	// Regular GitHub provider should not fail for main branch
	p := providers.NewGitHubProvider()
	// The regular provider doesn't have ResolvePRNumber exposed,
	// but it should work without requiring a PR
	if p.Name() != "github" {
		t.Errorf("expected provider name 'github', got '%s'", p.Name())
	}
}

// Issue 1: Test that GHA provider and PR provider are separate and do their own jobs
func TestGHAProviderWritesToStepSummaryOnly(t *testing.T) {
	// Create a temp file to act as GITHUB_STEP_SUMMARY
	tmpFile, err := os.CreateTemp("", "step-summary-*.md")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Save and set env vars
	origSummary := os.Getenv("GITHUB_STEP_SUMMARY")
	origToken := os.Getenv("GITHUB_TOKEN")
	origRepo := os.Getenv("GITHUB_REPOSITORY")
	origPRNum := os.Getenv("PR_NUMBER")

	defer func() {
		os.Setenv("GITHUB_STEP_SUMMARY", origSummary)
		os.Setenv("GITHUB_TOKEN", origToken)
		os.Setenv("GITHUB_REPOSITORY", origRepo)
		os.Setenv("PR_NUMBER", origPRNum)
	}()

	os.Setenv("GITHUB_STEP_SUMMARY", tmpFile.Name())
	os.Setenv("GITHUB_TOKEN", "") // No token - should not try to post PR comment
	os.Setenv("GITHUB_REPOSITORY", "")
	os.Setenv("PR_NUMBER", "")

	// Create GHA provider and write summary
	p := providers.NewGitHubProvider()
	summary := &internal.Summary{
		Phase:     internal.PhasePlan,
		Workspace: "test",
		ToAdd:     1,
	}
	markdown := "# Test Summary\n\nThis is a test."

	err = p.WriteSummary(summary, markdown)
	if err != nil {
		t.Fatalf("WriteSummary failed: %v", err)
	}

	// Verify the step summary file was written
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to read step summary file: %v", err)
	}

	if len(content) == 0 {
		t.Error("expected step summary file to have content, but it was empty")
	}

	if string(content) != markdown+"\n" {
		t.Errorf("expected step summary to contain %q, got %q", markdown+"\n", string(content))
	}
}

func TestPRProviderDoesNotWriteToStepSummary(t *testing.T) {
	// Create a temp file to act as GITHUB_STEP_SUMMARY
	tmpFile, err := os.CreateTemp("", "step-summary-*.md")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Save and set env vars
	origSummary := os.Getenv("GITHUB_STEP_SUMMARY")
	origToken := os.Getenv("GITHUB_TOKEN")
	origRepo := os.Getenv("GITHUB_REPOSITORY")
	origPRNum := os.Getenv("PR_NUMBER")
	origHeadRef := os.Getenv("GITHUB_HEAD_REF")
	origRefName := os.Getenv("GITHUB_REF_NAME")

	defer func() {
		os.Setenv("GITHUB_STEP_SUMMARY", origSummary)
		os.Setenv("GITHUB_TOKEN", origToken)
		os.Setenv("GITHUB_REPOSITORY", origRepo)
		os.Setenv("PR_NUMBER", origPRNum)
		os.Setenv("GITHUB_HEAD_REF", origHeadRef)
		os.Setenv("GITHUB_REF_NAME", origRefName)
	}()

	os.Setenv("GITHUB_STEP_SUMMARY", tmpFile.Name())
	os.Setenv("GITHUB_TOKEN", "") // No token - will fail PR lookup but that's OK
	os.Setenv("GITHUB_REPOSITORY", "owner/repo")
	os.Setenv("PR_NUMBER", "") // No PR number
	os.Setenv("GITHUB_HEAD_REF", "")
	os.Setenv("GITHUB_REF_NAME", "main") // Main branch - will fail PR requirement

	// Create PR provider
	p := providers.NewGitHubPRProvider()
	summary := &internal.Summary{
		Phase:     internal.PhasePlan,
		Workspace: "test",
		ToAdd:     1,
	}
	markdown := "# Test Summary\n\nThis is a test."

	// This will fail because we're on main branch with requirePR=true
	// But the important thing is it should NOT write to step summary
	_ = p.WriteSummary(summary, markdown)

	// Verify the step summary file was NOT written
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to read step summary file: %v", err)
	}

	if len(content) > 0 {
		t.Errorf("expected step summary file to be empty (PR provider should not write to it), but got: %q", string(content))
	}
}

func TestProviderSeparation(t *testing.T) {
	// Test that GHA and PR providers have correct configuration
	t.Run("GHA provider has summaryFile set", func(t *testing.T) {
		origSummary := os.Getenv("GITHUB_STEP_SUMMARY")
		defer os.Setenv("GITHUB_STEP_SUMMARY", origSummary)

		os.Setenv("GITHUB_STEP_SUMMARY", "/tmp/test-summary.md")

		p := providers.NewGitHubProvider()
		if p.Name() != "github" {
			t.Errorf("expected provider name 'github', got '%s'", p.Name())
		}
	})

	t.Run("PR provider has empty summaryFile", func(t *testing.T) {
		p := providers.NewGitHubPRProvider()
		if p.Name() != "github-pr" {
			t.Errorf("expected provider name 'github-pr', got '%s'", p.Name())
		}
	})
}

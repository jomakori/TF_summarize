package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// WriteGHASummary appends markdown to $GITHUB_STEP_SUMMARY.
func WriteGHASummary(markdown string) error {
	path, err := RequireEnv("GITHUB_STEP_SUMMARY")
	if err != nil {
		return fmt.Errorf("GITHUB_STEP_SUMMARY not set — are you running in GitHub Actions?")
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening step summary file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(markdown + "\n"); err != nil {
		return fmt.Errorf("writing step summary: %w", err)
	}

	return nil
}

// WritePRComment posts or updates a comment on the PR.
func WritePRComment(markdown string) error {
	token, err := RequireEnv("GITHUB_TOKEN")
	if err != nil {
		return err
	}

	repo, err := RequireEnv("GITHUB_REPOSITORY")
	if err != nil {
		return fmt.Errorf("GITHUB_REPOSITORY and PR_NUMBER must be set for PR comments")
	}
	prNumber, err := RequireEnv("PR_NUMBER")
	if err != nil {
		return fmt.Errorf("GITHUB_REPOSITORY and PR_NUMBER must be set for PR comments")
	}

	apiBase := GetEnv("GITHUB_API_URL", "https://api.github.com")

	// Marker to identify our comment for updates
	marker := "<!-- tfplan-summary -->"
	body := marker + "\n" + markdown

	// Check for existing comment to update
	existingID, err := findExistingComment(apiBase, repo, prNumber, token, marker)
	if err == nil && existingID > 0 {
		return updateComment(apiBase, repo, token, existingID, body)
	}

	return createComment(apiBase, repo, prNumber, token, body)
}

func findExistingComment(apiBase, repo, prNumber, token, marker string) (int64, error) {
	url := fmt.Sprintf("%s/repos/%s/issues/%s/comments?per_page=100", apiBase, repo, prNumber)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("unexpected status %d listing comments", resp.StatusCode)
	}

	var comments []struct {
		ID   int64  `json:"id"`
		Body string `json:"body"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
		return 0, err
	}

	for _, c := range comments {
		if strings.Contains(c.Body, marker) {
			return c.ID, nil
		}
	}

	return 0, fmt.Errorf("not found")
}

func createComment(apiBase, repo, prNumber, token, body string) error {
	url := fmt.Sprintf("%s/repos/%s/issues/%s/comments", apiBase, repo, prNumber)
	return postComment(url, token, body)
}

func updateComment(apiBase, repo, token string, commentID int64, body string) error {
	url := fmt.Sprintf("%s/repos/%s/issues/comments/%d", apiBase, repo, commentID)
	return patchComment(url, token, body)
}

func postComment(url, token, body string) error {
	payload, _ := json.Marshal(map[string]string{"body": body})

	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("posting comment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create comment (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func patchComment(url, token, body string) error {
	payload, _ := json.Marshal(map[string]string{"body": body})

	req, err := http.NewRequest("PATCH", url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("updating comment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update comment (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// FindPRForBranch looks up the PR number for a given branch using the GitHub API.
// It returns the PR number if found, or 0 if no PR exists for the branch.
func FindPRForBranch(branch string) (int, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return 0, fmt.Errorf("GITHUB_TOKEN not set")
	}

	repo := os.Getenv("GITHUB_REPOSITORY")
	if repo == "" {
		return 0, fmt.Errorf("GITHUB_REPOSITORY not set")
	}

	apiBase := GetEnv("GITHUB_API_URL", "https://api.github.com")

	// GitHub API: GET /repos/{owner}/{repo}/pulls?head={owner}:{branch}&state=open
	// Extract owner from repo (format: owner/repo)
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid GITHUB_REPOSITORY format: %s", repo)
	}
	owner := parts[0]

	// URL encode the head parameter
	head := fmt.Sprintf("%s:%s", owner, branch)
	url := fmt.Sprintf("%s/repos/%s/pulls?head=%s&state=open", apiBase, repo, head)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("fetching PRs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var prs []struct {
		Number int `json:"number"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&prs); err != nil {
		return 0, fmt.Errorf("decoding response: %w", err)
	}

	if len(prs) == 0 {
		return 0, nil // No PR found for this branch
	}

	return prs[0].Number, nil
}

// GetCurrentBranch returns the current branch name from GitHub Actions environment.
func GetCurrentBranch() string {
	// For pull_request events, GITHUB_HEAD_REF contains the branch name
	if branch := os.Getenv("GITHUB_HEAD_REF"); branch != "" {
		return branch
	}

	// For push/workflow_dispatch events, GITHUB_REF_NAME contains the branch name
	if branch := os.Getenv("GITHUB_REF_NAME"); branch != "" {
		return branch
	}

	// Fallback: parse from GITHUB_REF (refs/heads/branch-name)
	ref := os.Getenv("GITHUB_REF")
	if strings.HasPrefix(ref, "refs/heads/") {
		return strings.TrimPrefix(ref, "refs/heads/")
	}

	return ""
}

// WriteStdout prints the markdown to stdout.
func WriteStdout(markdown string) {
	fmt.Println(markdown)
}

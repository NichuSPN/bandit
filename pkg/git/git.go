package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"bandit/pkg/engine"
)

type GitManager struct{}

func NewGitManager() *GitManager {
	return &GitManager{}
}

func (gm *GitManager) GetStatus() (string, error) {
	cmd := exec.Command("git", "status", "--short")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errStr := stderr.String()
		if strings.Contains(errStr, "not a git repository") {
			return "Workspace is not a Git repository. Run 'git init' or use Bandit's internal diff engine.", nil
		}
		return "", fmt.Errorf("`git status` failed: %s", errStr)
	}

	statusStr := strings.TrimSpace(stdout.String())
	if statusStr == "" {
		return "Git Status: Working tree clean (no uncommitted changes).", nil
	}
	return fmt.Sprintf("Git Status (modified/untracked files):\n%s", statusStr), nil
}

func (gm *GitManager) GetDiff(filePath string) (string, error) {
	args := []string{"diff"}
	if filePath != "" {
		args = append(args, filePath)
	}

	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errStr := stderr.String()
		if strings.Contains(errStr, "not a git repository") {
			return "Non-Git Diff Tracking: Workspace is not a Git repository. Bandit uses internal snapshot diffing to compare original vs modified files.", nil
		}
		return "", fmt.Errorf("`git diff` failed: %s", errStr)
	}

	diffStr := strings.TrimSpace(stdout.String())
	if diffStr == "" {
		return "Git Diff: No active uncommitted diff found.", nil
	}
	return fmt.Sprintf("Git Diff:\n%s", diffStr), nil
}

func (gm *GitManager) GetRecentCommits(limit int) (string, error) {
	cmd := exec.Command("git", "log", "-n", fmt.Sprintf("%d", limit), "--oneline")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("`git log` failed: %s", stderr.String())
	}

	return fmt.Sprintf("Recent Commits:\n%s", strings.TrimSpace(stdout.String())), nil
}

func ComputeTextDiff(oldContent, newContent, fileLabel string) string {
	return engine.ComputeDiff(oldContent, newContent, fileLabel)
}

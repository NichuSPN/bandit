package git_test

import (
	"strings"
	"testing"

	"bandit/pkg/git"
)

func TestGitStatusExecution(t *testing.T) {
	gm := git.NewGitManager()
	status, err := gm.GetStatus()
	if err != nil {
		t.Fatalf("unexpected error running GetStatus: %v", err)
	}
	if status == "" {
		t.Errorf("expected status output to not be empty")
	}
}

func TestNonGitUnifiedDiff(t *testing.T) {
	oldContent := "hello\nworld"
	newContent := "hello\ngo\nworld"

	diff := git.ComputeTextDiff(oldContent, newContent, "test.txt")
	if !strings.Contains(diff, "+go") {
		t.Errorf("expected diff to contain '+go', got:\n%s", diff)
	}
}

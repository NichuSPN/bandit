package context_test

import (
	"strings"
	"testing"

	"bandit/pkg/context"
	"bandit/pkg/skills"
)

func TestContextManagerPromptGeneration(t *testing.T) {
	cm := context.NewContextManager()
	cm.SetProblem("Fix logger duplication bug")
	cm.AddFile("pkg/logger.go")
	cm.AddNote("Identified duplicate emitter call")
	cm.AddSkill(skills.Skill{Name: "go-concurrency", Content: "Use channels safely"})

	prompt := cm.PrepareClaudePrompt("", nil)

	if !strings.Contains(prompt, "Fix logger duplication bug") {
		t.Errorf("expected prompt to contain problem description")
	}
	if !strings.Contains(prompt, "pkg/logger.go") {
		t.Errorf("expected prompt to contain relevant file")
	}
	if !strings.Contains(prompt, "Identified duplicate emitter call") {
		t.Errorf("expected prompt to contain note")
	}
	if !strings.Contains(prompt, "go-concurrency") {
		t.Errorf("expected prompt to contain skill name")
	}
}

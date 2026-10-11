package interactive_test

import (
	"strings"
	"testing"

	"bandit/pkg/interactive"
)

func TestAskQuestionsNonTTY(t *testing.T) {
	questions := []interactive.QuestionItem{
		{
			Question: "Which database should we use?",
			Options:  []string{"PostgreSQL", "SQLite", "MySQL"},
		},
	}

	result, err := interactive.AskQuestions(questions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "Question: Which database should we use?") {
		t.Errorf("expected result to contain question")
	}

	if !strings.Contains(result, "User Selected Answer: PostgreSQL") {
		t.Errorf("expected result to contain default first option in non-TTY mode, got: %s", result)
	}
}

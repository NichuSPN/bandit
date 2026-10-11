package agent

import (
	"context"
	"strings"
	"testing"

	"bandit/pkg/config"
	"bandit/pkg/model"
)

func TestConversationRollbackOnCancel(t *testing.T) {
	cfg := config.Config{}
	ag := NewAgent(cfg)

	initialLen := len(ag.Conversation)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	progressChan := make(chan model.AgentProgressEvent, 10)
	done := make(chan struct{})
	go func() {
		for range progressChan {
		}
		close(done)
	}()

	_, err := ag.ProcessUserMessageStreamingContext(ctx, "cancelled prompt", false, progressChan)
	close(progressChan)
	<-done

	if err == nil {
		t.Fatalf("expected error on cancelled context, got nil")
	}

	finalLen := len(ag.Conversation)
	if finalLen != initialLen {
		t.Errorf("expected conversation length to roll back to %d, got %d", initialLen, finalLen)
	}
}

func TestExecuteToolAskQuestions(t *testing.T) {
	cfg := config.Config{}
	ag := NewAgent(cfg)

	payload := `{"questions":[{"question":"Choose an engine","options":["Rust","Go"]}]}`
	result := ag.ExecuteTool("ask_questions", []byte(payload))

	if !strings.Contains(result, "Choose an engine") {
		t.Errorf("expected result to include question, got %s", result)
	}
	if !strings.Contains(result, "Rust") {
		t.Errorf("expected result to include default selected option, got %s", result)
	}
}

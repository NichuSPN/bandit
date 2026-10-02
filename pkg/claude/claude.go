package claude

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type ClaudeManager struct {
	DefaultModel string
}

func NewClaudeManager(defaultModel string) *ClaudeManager {
	return &ClaudeManager{
		DefaultModel: defaultModel,
	}
}

func (cm *ClaudeManager) ListAvailableModels() []string {
	claudePath, err := exec.LookPath("claude")
	if err == nil {
		cmd := exec.Command(claudePath, "-p", "List only available Claude model IDs as a line separated list. Output only model names.")
		out, execErr := cmd.Output()
		if execErr == nil && len(out) > 0 {
			lines := strings.Split(string(out), "\n")
			var models []string
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				trimmed = strings.TrimPrefix(trimmed, "- ")
				trimmed = strings.Trim(trimmed, "`*")
				if trimmed != "" && !strings.HasPrefix(trimmed, "Here") && !strings.Contains(trimmed, " ") {
					models = append(models, trimmed)
				}
			}
			if len(models) > 0 {
				return models
			}
		}
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey != "" {
		req, err := http.NewRequest("GET", "https://api.anthropic.com/v1/models", nil)
		if err == nil {
			req.Header.Set("x-api-key", apiKey)
			req.Header.Set("anthropic-version", "2023-06-01")
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				defer resp.Body.Close()
				var apiResp struct {
					Data []struct {
						ID string `json:"id"`
					} `json:"data"`
				}
				if json.NewDecoder(resp.Body).Decode(&apiResp) == nil && len(apiResp.Data) > 0 {
					var models []string
					for _, m := range apiResp.Data {
						models = append(models, m.ID)
					}
					return models
				}
			}
		}
	}

	return []string{
		"sonnet",
		"haiku",
		"opus",
		"claude-3-7-sonnet-latest",
		"claude-3-5-sonnet-latest",
		"claude-3-5-haiku-latest",
	}
}

func (cm *ClaudeManager) RunInteractiveClaude(taskPrompt string) error {
	fmt.Println("=======================================================")
	fmt.Println("   Bandit Escalation -> Claude CLI (Interactive)")
	fmt.Printf("   Model: %s\n", cm.DefaultModel)
	fmt.Println("=======================================================")
	fmt.Println()

	args := []string{}
	if cm.DefaultModel != "" {
		args = append(args, "--model", cm.DefaultModel)
	}
	args = append(args, taskPrompt)

	var claudePath string
	var lookErr error
	claudePath, lookErr = exec.LookPath("claude")
	if lookErr != nil {
		claudePath = "claude"
	}

	cmd := exec.Command(claudePath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()

	fmt.Println()
	fmt.Println("=======================================================")
	if err == nil {
		fmt.Println(" ✓ Claude execution finished successfully.")
	} else {
		fmt.Printf(" ! Claude session exited: %v\n", err)
	}
	fmt.Println("=======================================================")
	fmt.Println()

	return err
}

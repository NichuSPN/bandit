package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"bandit/pkg/config"
	"bandit/pkg/model"
)

type OllamaTagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

type OllamaChatRequest struct {
	Model     string          `json:"model"`
	Messages  []model.Message `json:"messages"`
	Stream    bool            `json:"stream"`
	Tools     []interface{}   `json:"tools,omitempty"`
	KeepAlive interface{}     `json:"keep_alive,omitempty"`
	Options   map[string]any  `json:"options,omitempty"`
}

type OllamaStreamChunk struct {
	Message *struct {
		Content   string `json:"content"`
		ToolCalls []struct {
			Function struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			} `json:"function"`
		} `json:"tool_calls"`
	} `json:"message"`
	Done bool `json:"done"`
}

type OllamaClient struct {
	config     config.ModelConfig
	httpClient *http.Client
}

func NewOllamaClient(cfg config.ModelConfig) *OllamaClient {
	return &OllamaClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 0, // Disable overall request deadline so long LLM streams aren't cut off
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 60 * time.Second,
				IdleConnTimeout:       90 * time.Second,
			},
		},
	}
}

func (c *OllamaClient) ListInstalledModels() []string {
	baseURL := strings.TrimRight(c.config.BaseURL, "/")
	resp, err := c.httpClient.Get(baseURL + "/api/tags")
	if err != nil {
		return []string{"qwen3:14b", "qwen3-coder", "llama3:8b"}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []string{"qwen3:14b", "qwen3-coder", "llama3:8b"}
	}

	var tags OllamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tags); err == nil && len(tags.Models) > 0 {
		var res []string
		for _, m := range tags.Models {
			res = append(res, m.Name)
		}
		return res
	}

	return []string{"qwen3:14b", "qwen3-coder", "llama3:8b"}
}

func GetReadToolSchemas() []interface{} {
	return []interface{}{
		map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "list_directory",
				"description": "List files and directories within allowed root directory",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{"type": "string", "description": "Directory path (supports ~)"},
					},
					"required": []string{"path"},
				},
			},
		},
		map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "read_file",
				"description": "Read file content within allowed root directory",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{"type": "string", "description": "File path (supports ~)"},
					},
					"required": []string{"path"},
				},
			},
		},
		map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "search_files",
				"description": "Search file content for a text query across repository",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query":      map[string]interface{}{"type": "string", "description": "Search text query"},
						"target_dir": map[string]interface{}{"type": "string", "description": "Optional search root path"},
					},
					"required": []string{"query"},
				},
			},
		},
		map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "git_status",
				"description": "Check git repository status and uncommitted changes",
				"parameters": map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
		map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "git_diff",
				"description": "Inspect uncommitted git diff in the repository",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"file_path": map[string]interface{}{"type": "string", "description": "Optional file path filter"},
					},
				},
			},
		},
		map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "read_skill",
				"description": "Read domain skill from ./.bandit/skills, ./.claude/skills, ~/.bandit/skills, ~/.claude/skills",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"skill_name": map[string]interface{}{"type": "string", "description": "Name of domain skill (e.g. rust or postgres)"},
					},
					"required": []string{"skill_name"},
				},
			},
		},
	}
}

func GetAvailableToolSchemas(allowWriteTools bool) []interface{} {
	schemas := GetReadToolSchemas()
	if allowWriteTools {
		schemas = append(schemas,
			map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        "write_file",
					"description": "Create or overwrite file content within allowed workspace boundaries",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"path":    map[string]interface{}{"type": "string", "description": "File path (supports ~)"},
							"content": map[string]interface{}{"type": "string", "description": "Complete new content for file"},
						},
						"required": []string{"path", "content"},
					},
				},
			},
			map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        "edit_file",
					"description": "Replace a specific substring block in a file with new content",
					"parameters": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"path":                map[string]interface{}{"type": "string", "description": "File path (supports ~)"},
							"target_content":      map[string]interface{}{"type": "string", "description": "Exact text block to find and replace"},
							"replacement_content": map[string]interface{}{"type": "string", "description": "New replacement content block"},
						},
						"required": []string{"path", "target_content", "replacement_content"},
					},
				},
			},
		)
	}
	return schemas
}

func (c *OllamaClient) StreamChatContext(ctx context.Context, messages []model.Message, allowWriteTools bool, progressChan chan<- model.AgentProgressEvent) (string, []model.ToolCall, error) {
	return c.streamChatInternal(ctx, messages, true, allowWriteTools, progressChan)
}

func (c *OllamaClient) StreamChat(messages []model.Message, allowWriteTools bool, progressChan chan<- model.AgentProgressEvent) (string, []model.ToolCall, error) {
	return c.streamChatInternal(context.Background(), messages, true, allowWriteTools, progressChan)
}

func (c *OllamaClient) StreamChatWithoutTools(messages []model.Message, progressChan chan<- model.AgentProgressEvent) (string, error) {
	content, _, err := c.streamChatInternal(context.Background(), messages, false, false, progressChan)
	return content, err
}

func (c *OllamaClient) streamChatInternal(ctx context.Context, messages []model.Message, enableTools bool, allowWriteTools bool, progressChan chan<- model.AgentProgressEvent) (string, []model.ToolCall, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	baseURL := strings.TrimRight(c.config.BaseURL, "/")

	var tools []interface{}
	if enableTools {
		tools = GetAvailableToolSchemas(allowWriteTools)
	}

	options := map[string]any{
		"num_ctx":     16384,
		"num_predict": -1,
	}
	if c.config.NumGPU >= 0 {
		options["num_gpu"] = c.config.NumGPU
	}

	keepAliveVal := c.config.KeepAlive
	if keepAliveVal == "" {
		keepAliveVal = "-1"
	}

	reqBody := OllamaChatRequest{
		Model:     c.config.Model,
		Messages:  messages,
		Stream:    true,
		Tools:     tools,
		KeepAlive: keepAliveVal,
		Options:   options,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/chat", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return "", nil, ctx.Err()
		}
		return "", nil, fmt.Errorf("connection to Ollama failed (%s): %w\n  👉 Check if Ollama is running ('ollama serve' or open the Ollama app)", baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return "", nil, fmt.Errorf("Ollama model '%s' not found on server.\n  👉 Run 'ollama pull %s' to download the model", c.config.Model, c.config.Model)
		}
		return "", nil, fmt.Errorf("Ollama API error HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	reader := bufio.NewReader(resp.Body)
	var fullContent strings.Builder
	var collectedToolCalls []model.ToolCall

	for {
		if ctx.Err() != nil {
			return "", nil, ctx.Err()
		}

		line, err := reader.ReadString('\n')
		line = strings.TrimSpace(line)

		if line != "" {
			var chunk OllamaStreamChunk
			if err := json.Unmarshal([]byte(line), &chunk); err == nil {
				if chunk.Message != nil {
					if chunk.Message.Content != "" {
						fullContent.WriteString(chunk.Message.Content)
						if progressChan != nil {
							progressChan <- model.AgentProgressEvent{
								Type:  model.EventContentChunk,
								Chunk: chunk.Message.Content,
							}
						}
					}

					for _, tc := range chunk.Message.ToolCalls {
						toolCall := model.ToolCall{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						}
						collectedToolCalls = append(collectedToolCalls, toolCall)
					}
				}
			}
		}

		if err != nil {
			if ctx.Err() != nil {
				return "", nil, ctx.Err()
			}
			if err == io.EOF {
				break
			}
			return "", nil, fmt.Errorf("stream read error: %w", err)
		}
	}

	if progressChan != nil {
		progressChan <- model.AgentProgressEvent{Type: model.EventFinished}
	}
	return fullContent.String(), collectedToolCalls, nil
}

func (c *OllamaClient) UnloadModel() error {
	baseURL := strings.TrimRight(c.config.BaseURL, "/")
	reqBody := map[string]interface{}{
		"model":      c.config.Model,
		"keep_alive": 0,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/chat", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func IsOOMError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "out of memory") ||
		strings.Contains(msg, "vram") ||
		strings.Contains(msg, "cuda") ||
		strings.Contains(msg, "metal") ||
		strings.Contains(msg, "failed to allocate") ||
		strings.Contains(msg, "runner process") ||
		strings.Contains(msg, "signal: killed") ||
		strings.Contains(msg, "exit status 137") ||
		strings.Contains(msg, "unexpected eof")
}

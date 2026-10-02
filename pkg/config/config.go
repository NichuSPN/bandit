package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type InvestigationMode string

const (
	ModeFast     InvestigationMode = "fast"
	ModeResearch InvestigationMode = "research"
	ModeDeep     InvestigationMode = "deep"
)

func (m InvestigationMode) MaxIterations() int {
	switch m {
	case ModeFast:
		return 5
	case ModeDeep:
		return 20
	default:
		return 10
	}
}

func (m InvestigationMode) Name() string {
	return string(m)
}

func (m InvestigationMode) Label() string {
	switch m {
	case ModeFast:
		return "Fast Reply (5 tool iterations)"
	case ModeDeep:
		return "Deep Research Mode (20 tool iterations)"
	default:
		return "Research Mode (10 tool iterations)"
	}
}

func ParseInvestigationMode(s string) (InvestigationMode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "fast", "fast reply":
		return ModeFast, true
	case "research", "research mode":
		return ModeResearch, true
	case "deep", "deep research":
		return ModeDeep, true
	default:
		return ModeResearch, false
	}
}

type ModelConfig struct {
	Provider     string            `json:"provider"`
	Model        string            `json:"model"`
	ClaudeModel  string            `json:"claude_model"`
	BaseURL      string            `json:"base_url"`
	Mode         InvestigationMode `json:"mode"`
	SystemPrompt string            `json:"system_prompt,omitempty"`
}

func DefaultModelConfig() ModelConfig {
	return ModelConfig{
		Provider:     "ollama",
		Model:        "qwen3:14b",
		ClaudeModel:  "claude-3-5-sonnet-latest",
		BaseURL:      "http://localhost:11434",
		Mode:         ModeResearch,
		SystemPrompt: "",
	}
}

func (c *Config) GetCustomSystemPrompt(cwd string) string {
	if cwd != "" {
		projPromptPath := filepath.Join(cwd, ".bandit", "system_prompt.txt")
		if data, err := os.ReadFile(projPromptPath); err == nil && len(strings.TrimSpace(string(data))) > 0 {
			return strings.TrimSpace(string(data))
		}
	}

	home, err := os.UserHomeDir()
	if err == nil {
		userPromptPath := filepath.Join(home, ".config", "bandit", "system_prompt.txt")
		if data, err := os.ReadFile(userPromptPath); err == nil && len(strings.TrimSpace(string(data))) > 0 {
			return strings.TrimSpace(string(data))
		}
	}

	if strings.TrimSpace(c.Model.SystemPrompt) != "" {
		return strings.TrimSpace(c.Model.SystemPrompt)
	}

	return ""
}

type FilesystemConfig struct {
	Roots []string `json:"roots"`
}

func DefaultFilesystemConfig() FilesystemConfig {
	return FilesystemConfig{
		Roots: []string{"~"},
	}
}

type Config struct {
	Model      ModelConfig      `json:"model"`
	Filesystem FilesystemConfig `json:"filesystem"`
}

func DefaultConfig() Config {
	return Config{
		Model:      DefaultModelConfig(),
		Filesystem: DefaultFilesystemConfig(),
	}
}

func GetConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", "bandit", "config.json")
}

func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func LoadConfig() Config {
	cfgPath := GetConfigPath()
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		cfg := DefaultConfig()
		_ = cfg.Save()
		return cfg
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig()
	}

	if cfg.Model.Model == "" {
		cfg.Model.Model = "qwen3:14b"
	}
	if cfg.Model.ClaudeModel == "" {
		cfg.Model.ClaudeModel = "sonnet"
		_ = cfg.Save()
	}
	if cfg.Model.BaseURL == "" {
		cfg.Model.BaseURL = "http://localhost:11434"
	}
	if cfg.Model.Mode == "" {
		cfg.Model.Mode = ModeResearch
	}
	if len(cfg.Filesystem.Roots) == 0 {
		cfg.Filesystem.Roots = []string{"~"}
	}

	return cfg
}

func (c *Config) Save() error {
	cfgPath := GetConfigPath()
	dir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cfgPath, data, 0644)
}

func (c *Config) GetAllowedRoots() []string {
	var roots []string
	for _, r := range c.Filesystem.Roots {
		abs := ExpandPath(r)
		if eval, err := filepath.Abs(abs); err == nil {
			roots = append(roots, eval)
		}
	}
	return roots
}

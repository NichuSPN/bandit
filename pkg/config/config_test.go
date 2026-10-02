package config_test

import (
	"testing"

	"bandit/pkg/config"
)

func TestConfigDefaultsAndExpansion(t *testing.T) {
	cfg := config.DefaultConfig()

	if cfg.Model.Provider != "ollama" {
		t.Errorf("expected provider 'ollama', got '%s'", cfg.Model.Provider)
	}
	if cfg.Model.Model != "qwen3:14b" {
		t.Errorf("expected model 'qwen3:14b', got '%s'", cfg.Model.Model)
	}

	roots := cfg.GetAllowedRoots()
	if len(roots) == 0 {
		t.Errorf("expected allowed roots to not be empty")
	}

	mode, ok := config.ParseInvestigationMode("deep")
	if !ok || mode != config.ModeDeep {
		t.Errorf("expected mode 'deep', got '%v'", mode)
	}
	if mode.MaxIterations() != 20 {
		t.Errorf("expected max iterations 20 for deep mode, got %d", mode.MaxIterations())
	}
}

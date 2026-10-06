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

	if cfg.Model.NumGPU != -1 {
		t.Errorf("expected NumGPU -1 by default, got %d", cfg.Model.NumGPU)
	}
	if cfg.Model.VRAMLimit != "off" {
		t.Errorf("expected VRAMLimit 'off' by default, got '%s'", cfg.Model.VRAMLimit)
	}
	if cfg.Model.KeepAlive != "-1" {
		t.Errorf("expected KeepAlive '-1' by default, got '%s'", cfg.Model.KeepAlive)
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

func TestHardwareAndVRAMSuggestion(t *testing.T) {
	mem := config.GetSystemMemoryInfo()
	if mem.TotalRAMGB <= 0 {
		t.Errorf("expected TotalRAMGB > 0, got %f", mem.TotalRAMGB)
	}

	// Test 8GB RAM recommendation
	rec8 := config.SuggestVRAMSetting(config.HardwareMemoryInfo{TotalRAMGB: 8.0}, "qwen3:14b")
	if rec8.SuggestedNumGPU != 12 || rec8.SuggestedVRAMLimit != "4GB" {
		t.Errorf("expected 12 layers / 4GB for 8GB RAM, got %d layers / %s", rec8.SuggestedNumGPU, rec8.SuggestedVRAMLimit)
	}

	// Test 16GB RAM recommendation
	rec16 := config.SuggestVRAMSetting(config.HardwareMemoryInfo{TotalRAMGB: 16.0}, "qwen3:14b")
	if rec16.SuggestedNumGPU != 24 || rec16.SuggestedVRAMLimit != "8GB" {
		t.Errorf("expected 24 layers / 8GB for 16GB RAM, got %d layers / %s", rec16.SuggestedNumGPU, rec16.SuggestedVRAMLimit)
	}

	// Test 32GB RAM recommendation
	rec32 := config.SuggestVRAMSetting(config.HardwareMemoryInfo{TotalRAMGB: 32.0}, "qwen3:14b")
	if rec32.SuggestedNumGPU != -1 || rec32.SuggestedVRAMLimit != "off" {
		t.Errorf("expected -1 layers / off for 32GB RAM, got %d layers / %s", rec32.SuggestedNumGPU, rec32.SuggestedVRAMLimit)
	}
}

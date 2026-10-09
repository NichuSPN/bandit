package repomap_test

import (
	"os"
	"path/filepath"
	"testing"

	"bandit/pkg/repomap"
)

func TestExtractSymbolsAndGenerateMap(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "repomap_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testGoFile := filepath.Join(tempDir, "main.go")
	goCode := `package main

type UserConfig struct {
	Name string
}

func ProcessUser(cfg UserConfig) error {
	return nil
}
`
	if err := os.WriteFile(testGoFile, []byte(goCode), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	symbols, err := repomap.ExtractSymbols(testGoFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}

	if len(symbols) < 3 {
		t.Errorf("expected at least 3 symbols (package, struct, func), got %d", len(symbols))
	}

	gen := repomap.NewRepoMapGenerator(tempDir, 1024)
	repoMap := gen.GenerateMap()

	if repoMap == "" {
		t.Errorf("expected non-empty repo map")
	}
}

package ollama_test

import (
	"errors"
	"fmt"
	"testing"

	"bandit/pkg/ollama"
)

func TestIsOOMError(t *testing.T) {
	tests := []struct {
		err      error
		expected bool
	}{
		{nil, false},
		{errors.New("connection refused"), false},
		{errors.New("Ollama API error HTTP 500: out of memory allocating 4096MB"), true},
		{errors.New("CUDA error: out of memory"), true},
		{errors.New("metal GPU allocation failed"), true},
		{errors.New("llama runner process killed with signal: killed"), true},
		{errors.New("exit status 137"), true},
		{errors.New("stream read error: unexpected EOF"), true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.err), func(t *testing.T) {
			res := ollama.IsOOMError(tt.err)
			if res != tt.expected {
				t.Errorf("IsOOMError(%v) = %v; want %v", tt.err, res, tt.expected)
			}
		})
	}
}

func TestParseKeepAlive(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{"-1", -1},
		{"", -1},
		{"0", 0},
		{"300", 300},
		{"5m", "5m"},
		{"1h", "1h"},
	}

	for _, tt := range tests {
		res := ollama.ParseKeepAlive(tt.input)
		if res != tt.expected {
			t.Errorf("ParseKeepAlive(%q) = %v (%T); want %v (%T)", tt.input, res, res, tt.expected, tt.expected)
		}
	}
}

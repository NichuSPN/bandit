package cli

import (
	"bufio"
	"strings"
	"testing"
)

func TestReadInput_SingleLine(t *testing.T) {
	input := "hello world\n"
	reader := bufio.NewReader(strings.NewReader(input))

	firstLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := readMultiLineInput(reader, firstLine, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "hello world"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

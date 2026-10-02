package filesystem_test

import (
	"os"
	"testing"

	"bandit/pkg/config"
	"bandit/pkg/filesystem"
)

func TestFilesystemBoundaryValidation(t *testing.T) {
	cfg := config.DefaultConfig()
	fm := filesystem.NewFilesystemManager(&cfg)

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	if !fm.IsPathAllowed(cwd) {
		t.Errorf("expected cwd '%s' to be allowed", cwd)
	}

	if fm.IsPathAllowed("/etc/passwd") {
		t.Errorf("expected '/etc/passwd' to be forbidden")
	}
}

func TestFilesystemReadAndList(t *testing.T) {
	cfg := config.DefaultConfig()
	fm := filesystem.NewFilesystemManager(&cfg)

	testFile := "test_sample.tmp"
	_ = os.WriteFile(testFile, []byte("line1\nline2\nline3\n"), 0644)
	defer os.Remove(testFile)

	content, err := fm.ReadFile(testFile, 2)
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}
	if content == "" {
		t.Errorf("expected read content to not be empty")
	}

	listRes, err := fm.ListDirectory(".")
	if err != nil {
		t.Fatalf("unexpected list_directory error: %v", err)
	}
	if listRes == "" {
		t.Errorf("expected list_directory output to not be empty")
	}
}

func TestDotfileSearch(t *testing.T) {
	cfg := config.DefaultConfig()
	fm := filesystem.NewFilesystemManager(&cfg)

	dotFile := ".env.example"
	_ = os.WriteFile(dotFile, []byte("SECRET_KEY=bandit_test_secret_12345\n"), 0644)
	defer os.Remove(dotFile)

	res, err := fm.SearchFiles("bandit_test_secret_12345", ".")
	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}

	if !testing.Short() && res == "No occurrences of 'bandit_test_secret_12345' found." {
		t.Errorf("expected search to find match in .env.example dotfile, got: %s", res)
	}
}

func TestEditFileBlankLineRemoval(t *testing.T) {
	cfg := config.DefaultConfig()
	fm := filesystem.NewFilesystemManager(&cfg)

	testFile := "test_edit_formatting.tmp"
	initialContent := "func main() {\n\tprintln(\"step1\")\n\tprintln(\"step2\")\n\tprintln(\"step3\")\n}\n"
	_ = os.WriteFile(testFile, []byte(initialContent), 0644)
	defer os.Remove(testFile)

	// Edit to remove 2 lines: println("step2") and println("step3")
	_, err := fm.EditFile(testFile, "println(\"step2\")\n\tprintln(\"step3\")", "")
	if err != nil {
		t.Fatalf("unexpected EditFile error: %v", err)
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("unexpected ReadFile error: %v", err)
	}

	expected := "func main() {\n\tprintln(\"step1\")\n}\n"
	if string(data) != expected {
		t.Errorf("expected clean formatting without extra blank line, got:\n%q\nexpected:\n%q", string(data), expected)
	}
}

package filesystem

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bandit/pkg/config"
	"bandit/pkg/engine"
)

type FilesystemManager struct {
	allowedRoots []string
}

func NewFilesystemManager(cfg *config.Config) *FilesystemManager {
	return &FilesystemManager{
		allowedRoots: cfg.GetAllowedRoots(),
	}
}

func (fm *FilesystemManager) IsPathAllowed(path string) bool {
	expanded := config.ExpandPath(path)
	absPath, err := filepath.Abs(expanded)
	if err != nil {
		return false
	}

	evalPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		evalPath = absPath
	}

	for _, root := range fm.allowedRoots {
		evalRoot, err := filepath.EvalSymlinks(root)
		if err != nil {
			evalRoot = root
		}

		if strings.HasPrefix(evalPath, evalRoot) {
			return true
		}
	}

	return false
}

func (fm *FilesystemManager) ListDirectory(pathStr string) (string, error) {
	expanded := config.ExpandPath(pathStr)
	if !fm.IsPathAllowed(expanded) {
		return "", fmt.Errorf("security violation: path '%s' is outside allowed workspace boundaries", pathStr)
	}

	entries, err := os.ReadDir(expanded)
	if err != nil {
		return "", fmt.Errorf("failed to read directory '%s': %w", pathStr, err)
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Contents of directory '%s':", expanded))

	for _, entry := range entries {
		prefix := "[FILE]"
		if entry.IsDir() {
			prefix = "[DIR] "
		}
		lines = append(lines, fmt.Sprintf("  %s %s", prefix, entry.Name()))
	}

	return strings.Join(lines, "\n"), nil
}

func (fm *FilesystemManager) ReadFile(pathStr string, maxLines int) (string, error) {
	expanded := config.ExpandPath(pathStr)
	if !fm.IsPathAllowed(expanded) {
		return "", fmt.Errorf("security violation: path '%s' is outside allowed workspace boundaries", pathStr)
	}

	file, err := os.Open(expanded)
	if err != nil {
		return "", fmt.Errorf("failed to read file '%s': %w", pathStr, err)
	}
	defer file.Close()

	if maxLines <= 0 {
		maxLines = 500
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	totalLines := 0

	for scanner.Scan() {
		totalLines++
		if len(lines) < maxLines {
			lines = append(lines, scanner.Text())
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file '%s': %w", pathStr, err)
	}

	content := strings.Join(lines, "\n")
	if totalLines > maxLines {
		return fmt.Sprintf("%s\n\n... [Truncated: showing first %d of %d total lines]", content, maxLines, totalLines), nil
	}

	return content, nil
}

func (fm *FilesystemManager) SearchFiles(query string, targetDir string) (string, error) {
	searchPath := "."
	if targetDir != "" {
		searchPath = targetDir
	}
	expanded := config.ExpandPath(searchPath)

	if !fm.IsPathAllowed(expanded) {
		return "", fmt.Errorf("security violation: search path '%s' is outside allowed workspace boundaries", searchPath)
	}

	cwd, _ := os.Getwd()
	pref := config.LoadProjectPreferences(cwd)
	ignoresStr := strings.Join(pref.IgnoredPaths, ",")

	return engine.FastSearch(query, expanded, ignoresStr), nil
}

func CleanCodeFormatting(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")

	// Collapse 3 or more consecutive newlines into 2 newlines (single empty line gap)
	for strings.Contains(content, "\n\n\n") {
		content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	}

	// Strip trailing whitespace from lines
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}
	content = strings.Join(lines, "\n")

	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	return content
}

func (fm *FilesystemManager) WriteFile(pathStr string, content string) (string, error) {
	expanded := config.ExpandPath(pathStr)
	if !fm.IsPathAllowed(expanded) {
		return "", fmt.Errorf("security violation: path '%s' is outside allowed workspace boundaries", pathStr)
	}

	dir := filepath.Dir(expanded)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create parent directories for '%s': %w", pathStr, err)
	}

	cleaned := CleanCodeFormatting(content)

	if err := os.WriteFile(expanded, []byte(cleaned), 0644); err != nil {
		return "", fmt.Errorf("failed to write file '%s': %w", pathStr, err)
	}

	return fmt.Sprintf("Successfully written file '%s' (%d bytes).", pathStr, len(cleaned)), nil
}

func (fm *FilesystemManager) EditFile(pathStr string, targetContent string, replacementContent string) (string, error) {
	expanded := config.ExpandPath(pathStr)
	if !fm.IsPathAllowed(expanded) {
		return "", fmt.Errorf("security violation: path '%s' is outside allowed workspace boundaries", pathStr)
	}

	data, err := os.ReadFile(expanded)
	if err != nil {
		return "", fmt.Errorf("failed to read file '%s' for editing: %w", pathStr, err)
	}

	content := string(data)
	contentNormalized := strings.ReplaceAll(content, "\r\n", "\n")
	targetNormalized := strings.ReplaceAll(targetContent, "\r\n", "\n")
	replacementNormalized := strings.ReplaceAll(replacementContent, "\r\n", "\n")

	matchFound := false
	targetToReplace := targetNormalized

	if strings.Contains(contentNormalized, targetNormalized) {
		matchFound = true
	} else if strings.Contains(contentNormalized, targetNormalized+"\n") {
		matchFound = true
		targetToReplace = targetNormalized + "\n"
	} else if strings.Contains(contentNormalized, strings.TrimSpace(targetNormalized)) {
		matchFound = true
		targetToReplace = strings.TrimSpace(targetNormalized)
	}

	if !matchFound {
		return "", fmt.Errorf("target content not found in file '%s'", pathStr)
	}

	// For line deletions, automatically expand targetToReplace to include leading indentation and trailing newline
	if replacementNormalized == "" {
		if !strings.HasSuffix(targetToReplace, "\n") && strings.Contains(contentNormalized, targetToReplace+"\n") {
			targetToReplace = targetToReplace + "\n"
		}

		idx := strings.Index(contentNormalized, targetToReplace)
		if idx > 0 {
			lineStart := strings.LastIndex(contentNormalized[:idx], "\n")
			if lineStart == -1 {
				lineStart = 0
			} else {
				lineStart++
			}
			leadingIndent := contentNormalized[lineStart:idx]
			if strings.TrimSpace(leadingIndent) == "" && len(leadingIndent) > 0 {
				targetToReplace = leadingIndent + targetToReplace
			}
		}
	}

	updated := strings.Replace(contentNormalized, targetToReplace, replacementNormalized, 1)
	cleaned := CleanCodeFormatting(updated)

	if err := os.WriteFile(expanded, []byte(cleaned), 0644); err != nil {
		return "", fmt.Errorf("failed to write edited file '%s': %w", pathStr, err)
	}

	return fmt.Sprintf("Successfully edited file '%s'.", pathStr), nil
}

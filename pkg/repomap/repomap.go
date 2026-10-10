package repomap

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"bandit/pkg/engine"
)

type RepoMapGenerator struct {
	RootDir   string
	MaxTokens int
}

func NewRepoMapGenerator(rootDir string, maxTokens int) *RepoMapGenerator {
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	return &RepoMapGenerator{
		RootDir:   rootDir,
		MaxTokens: maxTokens,
	}
}

func (g *RepoMapGenerator) GenerateMap() string {
	res := engine.GenerateRepoMap(g.RootDir, g.MaxTokens)
	if res != "" {
		return res
	}
	return g.generateMapFallback()
}

func ExtractSymbols(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(filePath))
	var symbols []string

	scanner := bufio.NewScanner(file)
	lineNum := 0

	goPattern := regexp.MustCompile(`^(type\s+\w+\s+(struct|interface)|func\s+(\([^)]+\)\s+)?\w+|package\s+\w+)`)
	pyPattern := regexp.MustCompile(`^(class\s+\w+|def\s+\w+)`)
	tsPattern := regexp.MustCompile(`^(export\s+)?(class|interface|function|type|enum)\s+\w+`)
	rustPattern := regexp.MustCompile(`^(pub\s+)?(struct|enum|fn|trait|type|mod)\s+\w+`)
	cPattern := regexp.MustCompile(`^((typedef\s+)?struct|class|enum)\s+\w+`)

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "/*") {
			continue
		}

		matched := false
		switch ext {
		case ".go":
			if goPattern.MatchString(line) {
				matched = true
			}
		case ".py":
			if pyPattern.MatchString(line) {
				matched = true
			}
		case ".ts", ".tsx", ".js", ".jsx":
			if tsPattern.MatchString(line) {
				matched = true
			}
		case ".rs":
			if rustPattern.MatchString(line) {
				matched = true
			}
		case ".c", ".h", ".cpp", ".hpp":
			if cPattern.MatchString(line) {
				matched = true
			}
		}

		if matched {
			cleanLine := line
			if idx := strings.Index(cleanLine, "{"); idx != -1 {
				cleanLine = strings.TrimSpace(cleanLine[:idx])
			}
			symbols = append(symbols, fmt.Sprintf("L%d: %s", lineNum, cleanLine))
		}
	}

	return symbols, nil
}

func (g *RepoMapGenerator) generateMapFallback() string {
	var sb strings.Builder
	maxChars := g.MaxTokens * 4 // Approx 4 chars per token

	sb.WriteString("## Repository Outline (Tree-Sitter / Symbol Map)\n")

	ignoredDirs := map[string]bool{
		".git": true, "node_modules": true, "vendor": true, "target": true,
		"bin": true, "dist": true, "build": true, ".bandit": true, ".claude": true,
	}

	supportedExts := map[string]bool{
		".go": true, ".rs": true, ".py": true, ".ts": true, ".tsx": true,
		".js": true, ".jsx": true, ".c": true, ".cpp": true, ".h": true, ".hpp": true,
	}

	_ = filepath.Walk(g.RootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(g.RootDir, path)
		if err != nil {
			relPath = path
		}

		if info.IsDir() {
			if ignoredDirs[info.Name()] || (strings.HasPrefix(info.Name(), ".") && info.Name() != ".") {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !supportedExts[ext] {
			return nil
		}

		if sb.Len() >= maxChars {
			return filepath.SkipDir
		}

		symbols, err := ExtractSymbols(path)
		if err != nil || len(symbols) == 0 {
			return nil
		}

		var fileBlock strings.Builder
		fileBlock.WriteString(fmt.Sprintf("\n### %s\n", relPath))
		for _, sym := range symbols {
			fileBlock.WriteString(fmt.Sprintf("  - %s\n", sym))
		}

		if sb.Len()+fileBlock.Len() > maxChars {
			sb.WriteString("\n... [Repository Map truncated to stay within VRAM/token budget] ...\n")
			return filepath.SkipDir
		}

		sb.WriteString(fileBlock.String())
		return nil
	})

	return sb.String()
}

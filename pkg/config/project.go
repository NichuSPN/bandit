package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type ProjectPreferences struct {
	IgnoredPaths       []string `json:"ignored_paths"`
	IncludeHiddenFiles bool     `json:"include_hidden_files"`
	RespectGitignore   bool     `json:"respect_gitignore"`
}

func DefaultProjectPreferences() ProjectPreferences {
	return ProjectPreferences{
		IgnoredPaths: []string{
			".git",
			"node_modules",
			"target",
			"dist",
			"build",
			"venv",
			"backup_go",
		},
		IncludeHiddenFiles: true,
		RespectGitignore:   true,
	}
}

func GetProjectPreferencesPath(cwd string) string {
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			cwd = "."
		}
	}
	return filepath.Join(cwd, ".bandit", "config", "preferences.json")
}

func LoadProjectPreferences(cwd string) ProjectPreferences {
	prefPath := GetProjectPreferencesPath(cwd)
	data, err := os.ReadFile(prefPath)
	if err != nil {
		pref := DefaultProjectPreferences()
		_ = SaveProjectPreferences(cwd, pref)
		return pref
	}

	var pref ProjectPreferences
	if err := json.Unmarshal(data, &pref); err != nil {
		return DefaultProjectPreferences()
	}

	if len(pref.IgnoredPaths) == 0 {
		pref.IgnoredPaths = DefaultProjectPreferences().IgnoredPaths
	}

	return pref
}

func SaveProjectPreferences(cwd string, pref ProjectPreferences) error {
	prefPath := GetProjectPreferencesPath(cwd)
	dir := filepath.Dir(prefPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(pref, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(prefPath, data, 0644)
}

func AddIgnoredPath(cwd string, targetPath string) (ProjectPreferences, error) {
	pref := LoadProjectPreferences(cwd)
	targetPath = strings.TrimSpace(targetPath)
	if targetPath == "" {
		return pref, nil
	}

	for _, p := range pref.IgnoredPaths {
		if strings.EqualFold(p, targetPath) {
			return pref, nil // Already ignored
		}
	}

	pref.IgnoredPaths = append(pref.IgnoredPaths, targetPath)
	err := SaveProjectPreferences(cwd, pref)
	return pref, err
}

func RemoveIgnoredPath(cwd string, targetPath string) (ProjectPreferences, error) {
	pref := LoadProjectPreferences(cwd)
	targetPath = strings.TrimSpace(targetPath)

	var newIgnored []string
	for _, p := range pref.IgnoredPaths {
		if !strings.EqualFold(p, targetPath) {
			newIgnored = append(newIgnored, p)
		}
	}

	pref.IgnoredPaths = newIgnored
	err := SaveProjectPreferences(cwd, pref)
	return pref, err
}

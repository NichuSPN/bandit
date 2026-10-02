package skills

import (
	"os"
	"path/filepath"
	"strings"

	"bandit/pkg/config"
)

type Skill struct {
	Name    string
	Content string
	Path    string
}

type SkillsManager struct {
	searchRoots []string
}

func NewSkillsManager() *SkillsManager {
	roots := []string{
		"./.bandit/skills",
		"./.claude/skills",
		"~/.bandit/skills",
		"~/.claude/skills",
	}

	var expandedRoots []string
	for _, r := range roots {
		expandedRoots = append(expandedRoots, config.ExpandPath(r))
	}

	return &SkillsManager{
		searchRoots: expandedRoots,
	}
}

func (sm *SkillsManager) ListSkills() []Skill {
	var discovered []Skill
	seen := make(map[string]bool)

	for _, root := range sm.searchRoots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				// Skill directory containing SKILL.md
				skillName := entry.Name()
				skillPath := filepath.Join(root, skillName, "SKILL.md")
				if content, err := os.ReadFile(skillPath); err == nil {
					if !seen[skillName] {
						seen[skillName] = true
						discovered = append(discovered, Skill{
							Name:    skillName,
							Content: string(content),
							Path:    skillPath,
						})
					}
				}
			} else if strings.HasSuffix(entry.Name(), ".md") {
				// Skill markdown file directly in directory
				skillName := strings.TrimSuffix(entry.Name(), ".md")
				skillPath := filepath.Join(root, entry.Name())
				if content, err := os.ReadFile(skillPath); err == nil {
					if !seen[skillName] {
						seen[skillName] = true
						discovered = append(discovered, Skill{
							Name:    skillName,
							Content: string(content),
							Path:    skillPath,
						})
					}
				}
			}
		}
	}

	return discovered
}

func (sm *SkillsManager) ReadSkill(name string) (*Skill, bool) {
	cleanName := strings.TrimSuffix(name, ".md")
	skills := sm.ListSkills()
	for _, s := range skills {
		if strings.EqualFold(s.Name, cleanName) {
			return &s, true
		}
	}
	return nil, false
}

package context

import (
	"fmt"
	"strings"

	"bandit/pkg/model"
	"bandit/pkg/skills"
)

type ContextManager struct {
	ProblemDescription string
	RelevantFiles      map[string]bool
	RelevantSkills     []skills.Skill
	GitStatusSummary   string
	GitDiffSummary     string
	InvestigationNotes []string
}

func NewContextManager() *ContextManager {
	return &ContextManager{
		RelevantFiles: make(map[string]bool),
	}
}

func (cm *ContextManager) SetProblem(problem string) {
	cm.ProblemDescription = problem
}

func (cm *ContextManager) AddFile(path string) {
	if cm.RelevantFiles == nil {
		cm.RelevantFiles = make(map[string]bool)
	}
	cm.RelevantFiles[path] = true
}

func (cm *ContextManager) AddSkill(skill skills.Skill) {
	for _, s := range cm.RelevantSkills {
		if s.Name == skill.Name {
			return
		}
	}
	cm.RelevantSkills = append(cm.RelevantSkills, skill)
}

func (cm *ContextManager) AddNote(note string) {
	cm.InvestigationNotes = append(cm.InvestigationNotes, note)
}

func (cm *ContextManager) Clear() {
	cm.ProblemDescription = ""
	cm.RelevantFiles = make(map[string]bool)
	cm.RelevantSkills = nil
	cm.GitStatusSummary = ""
	cm.GitDiffSummary = ""
	cm.InvestigationNotes = nil
}

func (cm *ContextManager) PrepareClaudePrompt(overrideTask string, conversation []model.Message) string {
	taskTitle := overrideTask
	if taskTitle == "" {
		if cm.ProblemDescription == "" {
			taskTitle = "Refactor and fix codebase issue"
		} else {
			taskTitle = cm.ProblemDescription
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Task: %s\n\n", taskTitle))

	sb.WriteString("## Context & Investigation Summary\n")
	if len(cm.InvestigationNotes) > 0 {
		for _, note := range cm.InvestigationNotes {
			sb.WriteString(fmt.Sprintf("- %s\n", note))
		}
	} else {
		sb.WriteString("Bandit inspected repository structure and identified target files.\n")
	}
	sb.WriteString("\n")

	if len(conversation) > 0 {
		sb.WriteString("## Full Session Conversation & Investigation Log\n")
		for _, msg := range conversation {
			if msg.Role == model.RoleSystem {
				continue
			}
			if msg.Role == model.RoleUser {
				sb.WriteString(fmt.Sprintf("**User**: %s\n\n", msg.Content))
			} else if msg.Role == model.RoleAssistant {
				if msg.Content != "" {
					sb.WriteString(fmt.Sprintf("**Assistant**: %s\n\n", msg.Content))
				}
				for _, tc := range msg.ToolCalls {
					sb.WriteString(fmt.Sprintf("  *Tool Executed*: `%s(%s)`\n", tc.Name, string(tc.Arguments)))
				}
			} else if msg.Role == model.RoleTool {
				firstLines := msg.Content
				lines := strings.Split(msg.Content, "\n")
				if len(lines) > 6 {
					firstLines = strings.Join(lines[:6], "\n") + "\n... [truncated]"
				}
				sb.WriteString(fmt.Sprintf("  *Tool Output (%s)*:\n```\n%s\n```\n\n", msg.Name, firstLines))
			}
		}
	}

	if len(cm.RelevantFiles) > 0 {
		sb.WriteString("## Discovered Relevant Files\n")
		for file := range cm.RelevantFiles {
			sb.WriteString(fmt.Sprintf("- %s\n", file))
		}
		sb.WriteString("\n")
	}

	if cm.GitStatusSummary != "" {
		sb.WriteString("## Git Status\n")
		sb.WriteString(cm.GitStatusSummary)
		sb.WriteString("\n\n")
	}

	if cm.GitDiffSummary != "" {
		sb.WriteString("## Current Git Diff\n")
		sb.WriteString(cm.GitDiffSummary)
		sb.WriteString("\n\n")
	}

	if len(cm.RelevantSkills) > 0 {
		sb.WriteString("## Loaded Domain Skills & Project Conventions\n")
		for _, s := range cm.RelevantSkills {
			sb.WriteString(fmt.Sprintf("### Skill: %s.md\n%s\n\n", s.Name, s.Content))
		}
	}

	sb.WriteString("## Instructions for Claude\n")
	sb.WriteString("1. Inspect the target files listed above and implement the requested changes.\n")
	sb.WriteString("2. Follow all project conventions, clean formatting, error handling, and test requirements.\n")
	sb.WriteString("3. Provide clear explanation of your modifications.\n")

	return sb.String()
}

func (cm *ContextManager) GetSummary() string {
	prob := "None set"
	if cm.ProblemDescription != "" {
		prob = cm.ProblemDescription
	}

	return fmt.Sprintf(
		"Problem Description: %s\nRelevant Files Count: %d\nLoaded Domain Skills: %d\nInvestigation Steps Count: %d",
		prob,
		len(cm.RelevantFiles),
		len(cm.RelevantSkills),
		len(cm.InvestigationNotes),
	)
}

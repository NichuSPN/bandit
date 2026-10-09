package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"bandit/pkg/claude"
	"bandit/pkg/config"
	bcontext "bandit/pkg/context"
	"bandit/pkg/filesystem"
	"bandit/pkg/git"
	"bandit/pkg/model"
	"bandit/pkg/ollama"
	"bandit/pkg/repomap"
	"bandit/pkg/skills"
)

type Agent struct {
	Config       config.Config
	Context      *bcontext.ContextManager
	Conversation []model.Message
	FS           *filesystem.FilesystemManager
	Git          *git.GitManager
	Skills       *skills.SkillsManager
	Ollama       *ollama.OllamaClient
	Claude       *claude.ClaudeManager
}

func NewAgent(cfg config.Config) *Agent {
	fsMgr := filesystem.NewFilesystemManager(&cfg)
	gitMgr := git.NewGitManager()
	skillsMgr := skills.NewSkillsManager()
	ollamaClient := ollama.NewOllamaClient(cfg.Model)
	claudeMgr := claude.NewClaudeManager(cfg.Model.ClaudeModel)

	repoMapGen := repomap.NewRepoMapGenerator(".", 1024)
	repoMapStr := repoMapGen.GenerateMap()

	discoveredSkills := skillsMgr.ListSkills()
	var skillNames []string
	for _, s := range discoveredSkills {
		skillNames = append(skillNames, s.Name+".md")
	}
	skillsListStr := "None currently loaded"
	if len(skillNames) > 0 {
		skillsListStr = strings.Join(skillNames, ", ")
	}

	systemPromptText := fmt.Sprintf(`You are Bandit, an autonomous local terminal AI agent for software developers.

YOUR MISSION:
When the user asks a question, gives a task, or asks to find a term/symbol, YOU MUST IMMEDIATELY INVESTIGATE IT YOURSELF by calling your tools.
NEVER output tutorials, step-by-step guides, or text explaining how the user can check or search the codebase.
ALWAYS execute search or file reading tools yourself in your VERY FIRST turn.

%s

RULES FOR TOOL USE & INVESTIGATION:
1. MANDATORY SEARCHING: If the user asks where a symbol, term, function, or string (e.g. "firehose", "auth", "database") is used or located, check the Repository Outline above and call `+"`search_files`"+` or `+"`read_file`"+`.
2. EXPLORATION: Start investigations in the current working directory ("."). Use `+"`list_directory(\".\")`"+` to explore directories and `+"`read_file`"+` to inspect primary source files.
3. NO PERMISSION NEEDED: Do not ask the user for permission or present generic step-by-step guides. Call the tools directly.
4. DISCOVERED DOMAIN SKILLS: [%s] - Use `+"`read_skill`"+` to load domain skills.
5. READ-ONLY IN CHAT: Standard chat interaction is STRICTLY READ-ONLY. Use tool results to locate exact files, lines, and content, then synthesize a concrete answer backed by exact source file line numbers.
6. APPLYING EDITS: File modification tools are ONLY enabled during /local or /claude execution.`, repoMapStr, skillsListStr)

	customPrompt := cfg.GetCustomSystemPrompt(".")
	if customPrompt != "" {
		systemPromptText = customPrompt
	}

	systemPrompt := model.NewSystemMessage(systemPromptText)

	return &Agent{
		Config:       cfg,
		Context:      bcontext.NewContextManager(),
		Conversation: []model.Message{systemPrompt},
		FS:           fsMgr,
		Git:          gitMgr,
		Skills:       skillsMgr,
		Ollama:       ollamaClient,
		Claude:       claudeMgr,
	}
}

func (a *Agent) GetModelInfo() (string, string) {
	return a.Config.Model.Model, a.Config.Model.ClaudeModel
}

func (a *Agent) UpdateLocalModel(newModel string) error {
	a.Config.Model.Model = newModel
	a.Ollama = ollama.NewOllamaClient(a.Config.Model)
	return a.Config.Save()
}

func (a *Agent) UpdateClaudeModel(newClaude string) error {
	a.Config.Model.ClaudeModel = newClaude
	a.Claude = claude.NewClaudeManager(newClaude)
	return a.Config.Save()
}

func (a *Agent) UpdateMode(newMode config.InvestigationMode) error {
	a.Config.Model.Mode = newMode
	return a.Config.Save()
}

func (a *Agent) UpdateVRAMConfig(numGPU int, vramLimit string) error {
	a.Config.Model.NumGPU = numGPU
	a.Config.Model.VRAMLimit = vramLimit
	a.Ollama = ollama.NewOllamaClient(a.Config.Model)
	return a.Config.Save()
}

func (a *Agent) UnloadModel() error {
	return a.Ollama.UnloadModel()
}

func (a *Agent) GetModelParameterSize(modelName string) float64 {
	return a.Ollama.GetModelParameterSize(modelName)
}

func (a *Agent) ResetConversation() {
	a.Context.Clear()
	if len(a.Conversation) > 0 {
		a.Conversation = a.Conversation[:1] // Keep system prompt
	}
}

func (a *Agent) ListInstalledOllamaModels() []string {
	return a.Ollama.ListInstalledModels()
}

func (a *Agent) ListClaudeModels() []string {
	return a.Claude.ListAvailableModels()
}

func (a *Agent) ListSkills() []string {
	var names []string
	for _, s := range a.Skills.ListSkills() {
		names = append(names, s.Name)
	}
	return names
}

func (a *Agent) GetContextSummary() string {
	return a.Context.GetSummary()
}

func (a *Agent) ProcessUserMessageStreaming(userInput string, allowWriteTools bool, progressChan chan<- model.AgentProgressEvent) (string, error) {
	return a.ProcessUserMessageStreamingContext(context.Background(), userInput, allowWriteTools, progressChan)
}

func truncateToolOutput(output string, maxLines int) string {
	lines := strings.Split(output, "\n")
	if len(lines) <= maxLines {
		return output
	}
	head := lines[:maxLines/2]
	tail := lines[len(lines)-maxLines/2:]
	omitted := len(lines) - maxLines
	return fmt.Sprintf("%s\n... [%d lines omitted to conserve VRAM/context memory] ...\n%s",
		strings.Join(head, "\n"), omitted, strings.Join(tail, "\n"))
}

func (a *Agent) PruneConversationIfNeeded() {
	// Keep system prompt (index 0) and max recent 10 messages
	maxMessages := 11
	if len(a.Conversation) > maxMessages {
		sysPrompt := a.Conversation[0]
		recent := a.Conversation[len(a.Conversation)-(maxMessages-1):]
		pruned := make([]model.Message, 0, maxMessages)
		pruned = append(pruned, sysPrompt)
		pruned = append(pruned, recent...)
		a.Conversation = pruned
	}
}

func (a *Agent) ProcessUserMessageStreamingContext(ctx context.Context, userInput string, allowWriteTools bool, progressChan chan<- model.AgentProgressEvent) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	initialConvLen := len(a.Conversation)
	a.Context.SetProblem(userInput)
	a.Conversation = append(a.Conversation, model.NewUserMessage(userInput))

	defer func() {
		if ctx.Err() != nil {
			if len(a.Conversation) > initialConvLen {
				a.Conversation = a.Conversation[:initialConvLen]
			}
		}
	}()

	maxIterations := a.Config.Model.Mode.MaxIterations()
	var finalAnswer strings.Builder
	executedTools := make(map[string]bool)

	for iteration := 0; iteration < maxIterations; iteration++ {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		a.PruneConversationIfNeeded()

		if progressChan != nil {
			progressChan <- model.AgentProgressEvent{Type: model.EventThinking}
		}

		assistantContent, toolCalls, err := a.Ollama.StreamChatContext(ctx, a.Conversation, allowWriteTools, progressChan)
		if err != nil {
			return "", err
		}

		if len(toolCalls) == 0 {
			if assistantContent != "" {
				a.Conversation = append(a.Conversation, model.NewAssistantMessage(assistantContent))
				finalAnswer.WriteString(assistantContent)
			}
			break
		}

		a.Conversation = append(a.Conversation, model.NewAssistantMessageWithTools(assistantContent, toolCalls))

		newCalls := 0
		for _, tc := range toolCalls {
			toolSig := fmt.Sprintf("%s:%s", tc.Name, string(tc.Arguments))
			if executedTools[toolSig] {
				continue // Prevent infinite loop of identical tool calls
			}
			executedTools[toolSig] = true
			newCalls++

			if progressChan != nil {
				progressChan <- model.AgentProgressEvent{
					Type: model.EventToolCall,
					Name: tc.Name,
					Args: string(tc.Arguments),
				}
			}

			toolResult := a.ExecuteTool(tc.Name, tc.Arguments)
			compactResult := truncateToolOutput(toolResult, 40)

			if progressChan != nil {
				progressChan <- model.AgentProgressEvent{
					Type:   model.EventToolResult,
					Name:   tc.Name,
					Result: compactResult,
				}
			}

			a.Conversation = append(a.Conversation, model.NewToolMessage(tc.Name, compactResult))

			firstLine := toolResult
			if idx := strings.Index(toolResult, "\n"); idx != -1 {
				firstLine = toolResult[:idx]
			}
			a.Context.AddNote(fmt.Sprintf("Tool '%s' executed: %s", tc.Name, firstLine))
		}

		if newCalls == 0 {
			break
		}
	}

	if strings.TrimSpace(finalAnswer.String()) == "" {
		if progressChan != nil {
			progressChan <- model.AgentProgressEvent{Type: model.EventThinking}
		}

		synthPrompt := fmt.Sprintf("Based on all the tool investigation results gathered above, write a complete, clear, and detailed response to the user's request: '%s'", userInput)
		a.Conversation = append(a.Conversation, model.NewUserMessage(synthPrompt))

		finalSynthesis, err := a.Ollama.StreamChatWithoutTools(a.Conversation, progressChan)
		if err == nil && finalSynthesis != "" {
			a.Conversation = append(a.Conversation, model.NewAssistantMessage(finalSynthesis))
			finalAnswer.WriteString(finalSynthesis)
		}
	}

	return finalAnswer.String(), nil
}

func (a *Agent) ExecuteTool(name string, args json.RawMessage) string {
	var params map[string]interface{}
	_ = json.Unmarshal(args, &params)

	getString := func(key string) string {
		if val, ok := params[key].(string); ok {
			return val
		}
		return ""
	}

	switch name {
	case "list_directory":
		path := getString("path")
		if path == "" {
			path = "."
		}
		res, err := a.FS.ListDirectory(path)
		if err != nil {
			return err.Error()
		}
		return res

	case "read_file":
		path := getString("path")
		a.Context.AddFile(path)
		res, err := a.FS.ReadFile(path, 500)
		if err != nil {
			return err.Error()
		}
		return res

	case "search_files":
		query := getString("query")
		targetDir := getString("target_dir")
		res, err := a.FS.SearchFiles(query, targetDir)
		if err != nil {
			return err.Error()
		}
		return res

	case "git_status":
		status, err := a.Git.GetStatus()
		if err != nil {
			return err.Error()
		}
		a.Context.GitStatusSummary = status
		return status

	case "git_diff":
		filePath := getString("file_path")
		diff, err := a.Git.GetDiff(filePath)
		if err != nil {
			return err.Error()
		}
		a.Context.GitDiffSummary = diff
		return diff

	case "read_skill":
		skillName := getString("skill_name")
		if skill, ok := a.Skills.ReadSkill(skillName); ok {
			a.Context.AddSkill(*skill)
			return skill.Content
		}
		return fmt.Sprintf("Skill '%s' not found in defined skill directories.", skillName)

	case "write_file":
		path := getString("path")
		content := getString("content")
		res, err := a.FS.WriteFile(path, content)
		if err != nil {
			return err.Error()
		}
		a.Context.AddFile(path)
		return res

	case "edit_file":
		path := getString("path")
		targetContent := getString("target_content")
		replacementContent := getString("replacement_content")
		res, err := a.FS.EditFile(path, targetContent, replacementContent)
		if err != nil {
			return err.Error()
		}
		a.Context.AddFile(path)
		return res

	default:
		return fmt.Sprintf("Unknown tool '%s'", name)
	}
}

func (a *Agent) PrepareAndRunInteractiveClaude(overrideTask string) error {
	if status, err := a.Git.GetStatus(); err == nil {
		a.Context.GitStatusSummary = status
	}
	if diff, err := a.Git.GetDiff(""); err == nil {
		a.Context.GitDiffSummary = diff
	}

	synthInstruction := "Synthesize a complete, highly-detailed, and structured task prompt for Claude Code CLI based on all accumulated conversation history, tool results, file lines, and investigation findings above. State the exact target files, lines, and requested code changes."
	if overrideTask != "" {
		synthInstruction = fmt.Sprintf("Synthesize a complete, highly-detailed, and structured task prompt for Claude Code CLI for the task: '%s'. Include all relevant context, file paths, line references, and investigation findings above.", overrideTask)
	}

	synthConversation := append(a.Conversation, model.NewUserMessage(synthInstruction))
	claudeTaskPrompt, err := a.Ollama.StreamChatWithoutTools(synthConversation, nil)
	if err != nil || strings.TrimSpace(claudeTaskPrompt) == "" {
		claudeTaskPrompt = a.Context.PrepareClaudePrompt(overrideTask, a.Conversation)
	}

	return a.Claude.RunInteractiveClaude(claudeTaskPrompt)
}

func (a *Agent) PerformPostClaudeVerification() string {
	var report []string
	report = append(report, "=== Post-Claude Verification Report ===")

	if status, err := a.Git.GetStatus(); err == nil {
		report = append(report, fmt.Sprintf("1. Git Status:\n%s", status))
	}
	if diff, err := a.Git.GetDiff(""); err == nil {
		report = append(report, fmt.Sprintf("2. Uncommitted Diff:\n%s", diff))
	}

	return strings.Join(report, "\n\n")
}

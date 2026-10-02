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
When the user asks a question or gives a task, YOU MUST FULLY INVESTIGATE AND ANSWER IT YOURSELF using your tools. Do NOT ask the user for permission or present options like "Would you like me to read .env?". Instead, READ THE FILES YOURSELF using tools.

RULES FOR TOOL USE & INVESTIGATION:
1. ALWAYS start your investigation in the current working directory ("."). Do NOT list home directory "~/" unless explicitly asked.
2. Use `+"`list_directory(\".\")`"+` to explore project files.
3. Use `+"`read_file`"+` to inspect actual primary source files and configs (.env, Cargo.toml, package.json, src/, backend/, frontend/src/, etc.).
4. Use `+"`search_files`"+` to find database connections, imports, or keywords across project source code.
5. DISCOVERED DOMAIN SKILLS: [%s] - Use `+"`read_skill`"+` to load any relevant skill when working with related technologies or conventions.
6. HIGH-LEVEL ARCHITECTURE & QUERY ADHERENCE: Directly address the user's specific request. When asked about authentication or project structure, provide a high-level architectural explanation (e.g. "Okta OAuth2 / JWT Bearer Tokens") based on primary source files. Never list or analyze low-level obfuscated or minified JS class names from build bundles.
7. PLAN & REVIEW MODE (READ-ONLY IN CHAT): Standard chat interaction is STRICTLY READ-ONLY. Use read_file and search_files to investigate and propose exact modification plans with line numbers. Do NOT modify files on disk during standard chat. Instruct the user to run /local or /claude to execute the plan.
8. APPLYING EDITS: File modification tools are ONLY enabled during /local or /claude execution.`, skillsListStr)

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

			if progressChan != nil {
				progressChan <- model.AgentProgressEvent{
					Type:   model.EventToolResult,
					Name:   tc.Name,
					Result: toolResult,
				}
			}

			a.Conversation = append(a.Conversation, model.NewToolMessage(tc.Name, toolResult))

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

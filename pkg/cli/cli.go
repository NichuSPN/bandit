package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"bandit/pkg/agent"
	"bandit/pkg/config"
	"bandit/pkg/model"
	"bandit/pkg/ollama"
)

const (
	colorCyan   = "\x1b[36;1m"
	colorGreen  = "\x1b[32;1m"
	colorYellow = "\x1b[33;1m"
	colorRed    = "\x1b[31;1m"
	colorDim    = "\x1b[2m"
	colorBold   = "\x1b[1m"
	colorReset  = "\x1b[0m"
)

var (
	activeCancel context.CancelFunc
	activeCancelMu chan struct{}
)

func RunCLI(cfg config.Config) error {
	ag := agent.NewAgent(cfg)
	defer func() {
		fmt.Printf("%sUnloading model on exit...%s\r\n", colorDim, colorReset)
		_ = ag.UnloadModel()
	}()

	localModel, claudeModel := ag.GetModelInfo()

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "Unknown"
	} else {
		cwd = filepath.Clean(cwd)
	}

	fmt.Printf("%s=======================================================%s\r\n", colorCyan, colorReset)
	fmt.Printf("%s   Bandit Terminal AI Agent (Native Go CLI Mode)%s\r\n", colorCyan, colorReset)
	fmt.Printf("   %sLocal: %s | Claude: %s | Mode: %s | Workspace: %s%s\r\n",
		colorBold, localModel, claudeModel, ag.Config.Model.Mode.Name(), cwd, colorReset)
	fmt.Printf("%s   Type /help for slash commands or enter your prompt.%s\r\n", colorDim, colorReset)
	fmt.Printf("%s=======================================================%s\r\n\r\n", colorCyan, colorReset)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		for range sigChan {
			if activeCancel != nil {
				fmt.Printf("\r\n%s⏹ Interrupt received (Ctrl+C). Cancelling agent execution...%s\r\n\r\n", colorYellow, colorReset)
				os.Stdout.Sync()
				cancel := activeCancel
				activeCancel = nil
				cancel()
			} else {
				fmt.Printf("\r\n%s(Press Ctrl+D or type /exit to exit Bandit)%s\r\n", colorDim, colorReset)
				modeName := ag.Config.Model.Mode.Name()
				fmt.Printf("%sbandit (%s) > %s", colorYellow, modeName, colorReset)
				os.Stdout.Sync()
			}
		}
	}()

	reader := bufio.NewReader(os.Stdin)

	for {
		modeName := ag.Config.Model.Mode.Name()
		fmt.Printf("%sbandit (%s) > %s", colorYellow, modeName, colorReset)
		os.Stdout.Sync()

		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("\r\n%sExiting Bandit.%s\r\n", colorDim, colorReset)
			break
		}

		input, err := readMultiLineInput(reader, line, ag)
		if err != nil {
			fmt.Printf("\r\n%sExiting Bandit.%s\r\n", colorDim, colorReset)
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		if strings.HasPrefix(input, "/") {
			handleSlashCommand(ag, input, reader)
		} else {
			processUserPrompt(ag, input, false)
		}
	}

	return nil
}

func readMultiLineInput(reader *bufio.Reader, firstLine string, ag *agent.Agent) (string, error) {
	return strings.TrimRight(firstLine, "\r\n"), nil
}

func processUserPrompt(ag *agent.Agent, prompt string, allowWriteTools bool) {
	ctx, cancel := context.WithCancel(context.Background())
	activeCancel = cancel
	defer func() {
		activeCancel = nil
		cancel()
	}()

	progressChan := make(chan model.AgentProgressEvent, 100)
	isStreaming := false

	doneChan := make(chan struct{})

	go func() {
		defer close(doneChan)
		for event := range progressChan {
			switch event.Type {
			case model.EventThinking:
				if !isStreaming {
					fmt.Printf("  %s◌ Investigating codebase...%s\r\n", colorCyan, colorReset)
					os.Stdout.Sync()
				}
			case model.EventToolCall:
				fmt.Printf("  %s✓ Executing tool '%s' (%s)%s\r\n", colorDim, event.Name, event.Args, colorReset)
				os.Stdout.Sync()
			case model.EventToolResult:
				linesCount := len(strings.Split(event.Result, "\n"))
				if linesCount > 1 {
					fmt.Printf("  %s✓ Tool '%s' returned %d lines%s\r\n", colorDim, event.Name, linesCount, colorReset)
					os.Stdout.Sync()
				}
			case model.EventContentChunk:
				if !isStreaming {
					fmt.Printf("\r\n%sBandit:%s\r\n", colorGreen, colorReset)
					isStreaming = true
				}
				formatted := strings.ReplaceAll(event.Chunk, "\r\n", "\n")
				formatted = strings.ReplaceAll(formatted, "\n", "\r\n")
				fmt.Print(formatted)
				os.Stdout.Sync()
			case model.EventError:
				fmt.Printf("\r\n%sSystem Error: %s%s\r\n", colorRed, event.ErrorMsg, colorReset)
				os.Stdout.Sync()
			}
		}

		os.Stdout.Sync()
		fmt.Print("\r\n\r\n\r\n")
		os.Stdout.Sync()
	}()

	_, err := ag.ProcessUserMessageStreamingContext(ctx, prompt, allowWriteTools, progressChan)
	close(progressChan)
	<-doneChan

	if err != nil {
		if errors.Is(err, context.Canceled) || ctx.Err() != nil {
			fmt.Printf("%s✓ Agent execution cancelled. Ready for next command.%s\r\n\r\n", colorYellow, colorReset)
		} else if ollama.IsOOMError(err) {
			fmt.Print(FormatOOMErrorNotification(err))
		} else {
			fmt.Printf("\r\n%sExecution Error: %v%s\r\n\r\n", colorRed, err, colorReset)
		}
	}
}

func FormatOOMErrorNotification(err error) string {
	var sb strings.Builder
	sb.WriteString("\r\n" + colorRed + "============================================================" + colorReset + "\r\n")
	sb.WriteString(colorRed + "🚨 OUT OF MEMORY (OOM) / VRAM CRASH DETECTED" + colorReset + "\r\n")
	sb.WriteString(colorRed + "============================================================" + colorReset + "\r\n")
	sb.WriteString(fmt.Sprintf("Details: %v\r\n\r\n", err))
	sb.WriteString(colorYellow + "Suggested Actions:" + colorReset + "\r\n")
	sb.WriteString("  1. Run '" + colorCyan + "/vram suggest" + colorReset + "' to check recommended memory settings for your system.\r\n")
	sb.WriteString("  2. Set VRAM limit manually (e.g., '" + colorCyan + "/vram 4GB" + colorReset + "' or '" + colorCyan + "/vram 12" + colorReset + "').\r\n")
	sb.WriteString("  3. Switch to a lighter model (e.g., '" + colorCyan + "/model qwen2.5-coder:7b" + colorReset + "').\r\n")
	sb.WriteString("  4. Escalate heavy task execution to Claude CLI via '" + colorCyan + "/claude" + colorReset + "'.\r\n")
	sb.WriteString(colorRed + "============================================================" + colorReset + "\r\n\r\n")
	return sb.String()
}

func handleSlashCommand(ag *agent.Agent, cmdStr string, reader *bufio.Reader) {
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return
	}

	cmd := parts[0]

	switch cmd {
	case "/help":
		fmt.Printf("\r\n%sAvailable Slash Commands:%s\r\n", colorBold, colorReset)
		fmt.Printf("  %s%-18s%s - Show this help menu\r\n", colorCyan, "/help", colorReset)
		fmt.Printf("  %s%-18s%s - View or select investigation mode (fast/research/deep)\r\n", colorCyan, "/mode", colorReset)
		fmt.Printf("  %s%-18s%s - Show current local & Claude models\r\n", colorCyan, "/model", colorReset)
		fmt.Printf("  %s%-18s%s - List installed local Ollama models\r\n", colorCyan, "/model list", colorReset)
		fmt.Printf("  %s%-18s%s - List available Claude models\r\n", colorCyan, "/model claude list", colorReset)
		fmt.Printf("  %s%-18s%s - Inspect or configure VRAM & GPU limits (/vram suggest)\r\n", colorCyan, "/vram", colorReset)
		fmt.Printf("  %s%-18s%s - Enter multi-line paste mode (type '/end' to submit)\r\n", colorCyan, "/paste", colorReset)
		fmt.Printf("  %s%-18s%s - View/manage project ignored paths (./.bandit/config/preferences.json)\r\n", colorCyan, "/ignore", colorReset)
		fmt.Printf("  %s%-18s%s - Clear conversation history and context\r\n", colorCyan, "/clear", colorReset)
		fmt.Printf("  %s%-18s%s - List discovered domain skills across 4 locations\r\n", colorCyan, "/skills", colorReset)
		fmt.Printf("  %s%-18s%s - View accumulated session context\r\n", colorCyan, "/context", colorReset)
		fmt.Printf("  %s%-18s%s - Execute code modifications directly using local model\r\n", colorCyan, "/local", colorReset)
		fmt.Printf("  %s%-18s%s - Escalate task to interactive Claude CLI\r\n", colorCyan, "/claude", colorReset)
		fmt.Printf("  %s%-18s%s - Exit Bandit session\r\n\r\n", colorCyan, "/exit", colorReset)

	case "/vram":
		memInfo := config.GetSystemMemoryInfo()
		if len(parts) == 1 || (len(parts) == 2 && (parts[1] == "info" || parts[1] == "status")) {
			gpuStr := "Full Offload (-1)"
			if ag.Config.Model.NumGPU >= 0 {
				gpuStr = fmt.Sprintf("%d layers", ag.Config.Model.NumGPU)
			}
			fmt.Printf("\r\n%sVRAM & Memory Configuration:%s\r\n", colorBold, colorReset)
			fmt.Printf("  Host System Memory: %s%.1f GB RAM%s (%s/%s)\r\n", colorCyan, memInfo.TotalRAMGB, colorReset, memInfo.OSName, memInfo.Arch)
			fmt.Printf("  VRAM Usage Limit  : %s%s%s\r\n", colorCyan, ag.Config.Model.VRAMLimit, colorReset)
			fmt.Printf("  GPU Layer Offload : %s%s%s\r\n", colorCyan, gpuStr, colorReset)
			fmt.Printf("  Session Keep-Alive: %s%s (locked in VRAM during session)%s\r\n\r\n", colorCyan, ag.Config.Model.KeepAlive, colorReset)
			fmt.Printf("%sUsage:%s\r\n", colorBold, colorReset)
			fmt.Printf("  - %s/vram suggest%s          : Get suggested VRAM configuration based on hardware\r\n", colorCyan, colorReset)
			fmt.Printf("  - %s/vram <4GB|8GB|off>%s    : Limit VRAM usage\r\n", colorCyan, colorReset)
			fmt.Printf("  - %s/vram <num_layers>%s     : Specify exact GPU layer count (e.g., /vram 12)\r\n\r\n", colorCyan, colorReset)
		} else if len(parts) >= 2 && parts[1] == "suggest" {
			rec := config.SuggestVRAMSetting(memInfo, ag.Config.Model.Model)
			fmt.Printf("\r\n%sSystem Memory & VRAM Suggestion:%s\r\n", colorBold, colorReset)
			fmt.Printf("  Host RAM          : %s%.1f GB%s\r\n", colorCyan, memInfo.TotalRAMGB, colorReset)
			fmt.Printf("  Suggested VRAM    : %s%s%s\r\n", colorGreen, rec.SuggestedVRAMLimit, colorReset)
			gpuRec := "Full Offload (-1)"
			if rec.SuggestedNumGPU >= 0 {
				gpuRec = fmt.Sprintf("%d layers", rec.SuggestedNumGPU)
			}
			fmt.Printf("  Suggested Layers  : %s%s%s\r\n", colorGreen, gpuRec, colorReset)
			fmt.Printf("  Suggested Model   : %s%s%s\r\n", colorGreen, rec.SuggestedModel, colorReset)
			fmt.Printf("  Reason            : %s\r\n\r\n", rec.Reason)
			fmt.Printf("To apply recommendation:\r\n  Run '%s/vram %s%s'\r\n", colorCyan, rec.SuggestedVRAMLimit, colorReset)
			if rec.SuggestedModel != ag.Config.Model.Model {
				fmt.Printf("  Run '%s/model %s%s'\r\n", colorCyan, rec.SuggestedModel, colorReset)
			}
			fmt.Println()
		} else if len(parts) >= 2 {
			arg := strings.ToLower(parts[1])
			numGPU := -1
			vramLimit := "off"

			switch arg {
			case "off", "-1", "none", "unlimited":
				numGPU = -1
				vramLimit = "off"
			case "4gb", "4g":
				numGPU = 12
				vramLimit = "4GB"
			case "8gb", "8g":
				numGPU = 24
				vramLimit = "8GB"
			case "12gb", "12g":
				numGPU = 32
				vramLimit = "12GB"
			case "16gb", "16g":
				numGPU = 36
				vramLimit = "16GB"
			case "cpu", "0":
				numGPU = 0
				vramLimit = "CPU only"
			default:
				if n, err := strconv.Atoi(arg); err == nil {
					numGPU = n
					if n < 0 {
						numGPU = -1
						vramLimit = "off"
					} else {
						vramLimit = fmt.Sprintf("%d layers", n)
					}
				} else {
					vramLimit = parts[1]
					numGPU = 16
				}
			}

			if err := ag.UpdateVRAMConfig(numGPU, vramLimit); err == nil {
				fmt.Printf("%s✓ VRAM limit set to '%s' (num_gpu: %d)%s\r\n\r\n", colorGreen, vramLimit, numGPU, colorReset)
			} else {
				fmt.Printf("%s✕ Failed to update VRAM config: %v%s\r\n\r\n", colorRed, err, colorReset)
			}
		}

	case "/paste", "/multiline":
		fmt.Printf("%sEntering Multi-Line Paste Mode. Type or paste your prompt.\r\nType '/end' on its own line when finished:%s\r\n", colorCyan, colorReset)
		var lines []string
		for {
			fmt.Printf("%s... > %s", colorDim, colorReset)
			os.Stdout.Sync()
			next, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			trimmed := strings.TrimRight(next, "\r\n")
			if strings.TrimSpace(trimmed) == "/end" {
				break
			}
			lines = append(lines, trimmed)
		}
		fullPrompt := strings.Join(lines, "\n")
		if strings.TrimSpace(fullPrompt) != "" {
			processUserPrompt(ag, fullPrompt, false)
		}

	case "/ignore":
		cwd, _ := os.Getwd()
		if len(parts) == 1 || (len(parts) == 2 && parts[1] == "list") {
			pref := config.LoadProjectPreferences(cwd)
			fmt.Printf("\r\n%sProject Ignored Paths (./.bandit/config/preferences.json):%s\r\n", colorBold, colorReset)
			for _, p := range pref.IgnoredPaths {
				fmt.Printf("  - %s%s%s\r\n", colorCyan, p, colorReset)
			}
			fmt.Printf("\r\nTo add path: /ignore add <path>\r\nTo remove path: /ignore remove <path>\r\n\r\n")
		} else if len(parts) >= 3 && parts[1] == "add" {
			targetPath := parts[2]
			pref, err := config.AddIgnoredPath(cwd, targetPath)
			if err == nil {
				fmt.Printf("%s✓ Added '%s' to project ignored paths in ./.bandit/config/preferences.json%s\r\n\r\n", colorGreen, targetPath, colorReset)
			} else {
				fmt.Printf("%s✕ Failed to save project preference: %v%s\r\n\r\n", colorRed, err, colorReset)
			}
			_ = pref
		} else if len(parts) >= 3 && parts[1] == "remove" {
			targetPath := parts[2]
			pref, err := config.RemoveIgnoredPath(cwd, targetPath)
			if err == nil {
				fmt.Printf("%s✓ Removed '%s' from project ignored paths%s\r\n\r\n", colorGreen, targetPath, colorReset)
			} else {
				fmt.Printf("%s✕ Failed to save project preference: %v%s\r\n\r\n", colorRed, err, colorReset)
			}
			_ = pref
		}

	case "/mode":
		activeMode := ag.Config.Model.Mode
		if len(parts) == 1 || (len(parts) == 2 && parts[1] == "list") {
			fmt.Printf("\r\n%sInvestigation Modes:%s\r\n", colorBold, colorReset)
			modes := []config.InvestigationMode{config.ModeFast, config.ModeResearch, config.ModeDeep}
			for _, m := range modes {
				activeBadge := ""
				if m == activeMode {
					activeBadge = fmt.Sprintf(" %s(active)%s", colorGreen, colorReset)
				}
				fmt.Printf("  - %s%-10s%s : %s%s\r\n", colorCyan, m.Name(), colorReset, m.Label(), activeBadge)
			}
			fmt.Printf("\r\nTo switch mode: /mode <fast | research | deep>\r\n\r\n")
		} else if len(parts) >= 2 {
			if targetMode, ok := config.ParseInvestigationMode(parts[1]); ok {
				if err := ag.UpdateMode(targetMode); err == nil {
					fmt.Printf("%s✓ Mode set to '%s' (%s)%s\r\n\r\n", colorGreen, targetMode.Name(), targetMode.Label(), colorReset)
				} else {
					fmt.Printf("%s✕ Failed to save mode: %v%s\r\n\r\n", colorRed, err, colorReset)
				}
			} else {
				fmt.Printf("%s✕ Invalid mode '%s'. Choose from: fast, research, deep%s\r\n\r\n", colorRed, parts[1], colorReset)
			}
		}

	case "/model":
		if len(parts) == 1 {
			local, claude := ag.GetModelInfo()
			fmt.Printf("\r\n%sActive Models:%s\r\n  Local: %s%s%s\r\n  Claude: %s%s%s\r\n\r\n",
				colorBold, colorReset, colorCyan, local, colorReset, colorCyan, claude, colorReset)
		} else if len(parts) == 2 && parts[1] == "list" {
			models := ag.ListInstalledOllamaModels()
			fmt.Printf("\r\n%sInstalled Ollama Models:%s\r\n", colorBold, colorReset)
			for _, m := range models {
				activeBadge := ""
				if m == ag.Config.Model.Model {
					activeBadge = fmt.Sprintf(" %s(active)%s", colorGreen, colorReset)
				}
				fmt.Printf("  - %s%s%s\r\n", m, activeBadge, colorReset)
			}
			fmt.Printf("\r\nTo set local model: /model <model-name>\r\n\r\n")
		} else if len(parts) >= 2 && parts[1] == "claude" {
			if len(parts) == 2 || (len(parts) == 3 && parts[2] == "list") {
				models := ag.ListClaudeModels()
				fmt.Printf("\r\n%sAvailable Claude Models:%s\r\n", colorBold, colorReset)
				for _, m := range models {
					activeBadge := ""
					if strings.EqualFold(m, ag.Config.Model.ClaudeModel) {
						activeBadge = fmt.Sprintf(" %s(active)%s", colorGreen, colorReset)
					}
					fmt.Printf("  - %s%s%s\r\n", m, activeBadge, colorReset)
				}
				fmt.Printf("\r\nTo set Claude model: /model claude <model-name>\r\n\r\n")
			} else if len(parts) >= 3 {
				targetClaude := parts[2]
				if err := ag.UpdateClaudeModel(targetClaude); err == nil {
					fmt.Printf("%s✓ Claude model updated to '%s'%s\r\n\r\n", colorGreen, targetClaude, colorReset)
				} else {
					fmt.Printf("%s✕ Failed to update Claude model: %v%s\r\n\r\n", colorRed, err, colorReset)
				}
			}
		} else if len(parts) >= 2 {
			targetModel := parts[1]
			if err := ag.UpdateLocalModel(targetModel); err == nil {
				fmt.Printf("%s✓ Local model updated to '%s'%s\r\n\r\n", colorGreen, targetModel, colorReset)
			} else {
				fmt.Printf("%s✕ Failed to update local model: %v%s\r\n\r\n", colorRed, err, colorReset)
			}
		}

	case "/clear":
		ag.ResetConversation()
		fmt.Printf("%s✓ Conversation history and context reset.%s\r\n\r\n", colorGreen, colorReset)

	case "/skills":
		skillList := ag.ListSkills()
		if len(skillList) == 0 {
			fmt.Printf("%sNo domain skills found across search paths.%s\r\n\r\n", colorYellow, colorReset)
		} else {
			fmt.Printf("\r\n%sDiscovered Domain Skills (%d):%s\r\n", colorBold, len(skillList), colorReset)
			for _, s := range skillList {
				fmt.Printf("  - %s%s.md%s\r\n", colorCyan, s, colorReset)
			}
			fmt.Println()
		}

	case "/context":
		fmt.Printf("\r\n%sSession Context Summary:%s\r\n%s\r\n\r\n", colorBold, colorReset, ag.GetContextSummary())

	case "/claude":
		fmt.Printf("%sEscalating task to interactive Claude CLI session...%s\r\n", colorCyan, colorReset)
		err := ag.PrepareAndRunInteractiveClaude("")
		if err != nil {
			fmt.Printf("%s✕ Claude execution failed: %v%s\r\n\r\n", colorRed, err, colorReset)
		} else {
			verificationReport := ag.PerformPostClaudeVerification()
			fmt.Printf("\r\n%s%s%s\r\n\r\n", colorCyan, verificationReport, colorReset)
		}

	case "/local":
		taskPrompt := strings.TrimSpace(strings.TrimPrefix(cmdStr, "/local"))
		if taskPrompt == "" {
			taskPrompt = "Apply the necessary code modifications directly to the files based on the accumulated context."
		} else {
			taskPrompt = "Apply the following code changes directly to the files: " + taskPrompt
		}
		fmt.Printf("%sExecuting code modifications with local model (%s)...%s\r\n", colorCyan, ag.Config.Model.Model, colorReset)
		processUserPrompt(ag, taskPrompt, true)
		verificationReport := ag.PerformPostClaudeVerification()
		fmt.Printf("\r\n%s%s%s\r\n\r\n", colorCyan, verificationReport, colorReset)

	case "/exit":
		fmt.Printf("%sUnloading model and exiting Bandit. Goodbye!%s\r\n", colorDim, colorReset)
		_ = ag.UnloadModel()
		os.Exit(0)

	default:
		fmt.Printf("%s✕ Unknown command '%s'. Type /help for available commands.%s\r\n\r\n", colorRed, cmd, colorReset)
	}
}

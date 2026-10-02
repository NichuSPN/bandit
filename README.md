# 🏴‍☠️ Bandit Terminal AI Agent

[![CI & Tests](https://github.com/NichuSPN/bandit/actions/workflows/ci.yml/badge.svg)](https://github.com/NichuSPN/bandit/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Bandit** is a high-performance, autonomous local terminal AI coding agent built with a hybrid Go CLI and CGO Rust execution engine. It interfaces locally with Ollama models and seamlessly escalates complex tasks to the interactive Claude Code CLI.

---

## 🔥 Key Features

- **Strict Read-Only Chat Boundary**: Standard interactive chat is strictly read-only—allowing you to inspect codebase diffs, search files, and plan changes without modifying disk state.
- **Controlled Disk Mutations (`/local` & `/claude`)**: Disk modifications occur strictly when explicitly triggered via `/local` or `/claude`.
- **Local Model Escalation Synthesis**: Running `/claude` invokes your local LLM to inspect session history, line numbers, and findings, synthesizing an optimized prompt before launching Claude CLI.
- **Multi-Line Paste Mode (`/multiline`)**: Dedicated `/paste` or `/multiline` mode (finish with `/end`) allows entry of multi-line prompts and code blocks.
- **Custom Per-User & Project System Prompts**: Customize behavior per project (`./.bandit/system_prompt.txt`), globally (`~/.config/bandit/system_prompt.txt`), or via `config.json`.
- **Clean Formatting & Line Deletions**: Automatically collapses extra blank lines and cleanly trims indentation when deleting lines.
- **Domain Skills Auto-Discovery**: Automatically discovers skills across 4 locations (`./.bandit/skills`, `./.claude/skills`, `~/.bandit/skills`, `~/.claude/skills`).
- **Graceful Signal Interrupts (`Ctrl+C`)**: Interrupting an active step immediately cancels context and rolls back un-answered prompts from conversation history.

---

## 🚀 Quick Start & Installation (Build from Source)

### Prerequisites
- **Go**: Version `1.24+` ([golang.org](https://go.dev/dl/))
- **Rust (`cargo`)**: Install Rust via [rustup.rs](https://rustup.rs/):
  ```bash
  curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
  source $HOME/.cargo/env
  ```
- **GCC / MinGW**: Required for CGO compilation on Linux/Windows ([mingw-w64](https://www.mingw-w64.org/))

---

### 🍏 macOS & Linux Installation

```bash
# 1. Clone the repository
git clone https://github.com/NichuSPN/bandit.git
cd bandit

# 2. Build single executable & install globally
./build.sh

# 3. Launch Bandit anywhere!
bandit
```

`./build.sh` statically compiles the CGO Rust engine library, builds the `./bandit` executable, and automatically installs it to `~/.cargo/bin/bandit`.

---

### 🪟 Windows Installation

Open PowerShell:

```powershell
# 1. Clone the repository
git clone https://github.com/NichuSPN/bandit.git
cd bandit

# 2. Build local executable
.\build.ps1

# 3. Run Bandit
.\bandit.exe
```

Or run directly with Go:

```powershell
# Step 1: Compile Rust engine static library
cd bandit_engine
cargo build --release
cd ..

# Step 2: Run Go CLI directly
go run main.go
```

---

## 💻 Usage & Slash Commands

Launch Bandit in any project directory:

```bash
bandit
```

### Command Reference

| Slash Command | Description |
| :--- | :--- |
| `/mode` | View or select investigation mode (`fast`, `research`, `deep`) |
| `/model` | Show current local model and Claude settings |
| `/model list` | List installed Ollama models |
| `/multiline` (or `/paste`) | Enter multi-line paste mode (type `/end` on its own line to submit) |
| `/local <prompt>` | Execute code modifications directly using local LLM |
| `/claude <prompt>` | Synthesize prompt and escalate task to interactive Claude CLI |
| `/skills` | List discovered domain skills across project and user paths |
| `/context` | Inspect accumulated session context |
| `/ignore` | Manage project ignored paths (`./.bandit/config/preferences.json`) |
| `/clear` | Clear conversation history and reset context |
| `/help` | Show available slash commands |
| `/exit` | Exit Bandit |

---

## 🛠 Project Architecture

```
bandit/
├── main.go               # Entry point
├── build.sh              # 2-Step Hybrid Build script (Go + Rust)
├── build.ps1             # Windows PowerShell build script
├── bandit_engine/        # Core Rust engine (CGO static library)
│   ├── Cargo.toml
│   └── src/lib.rs
├── pkg/                  # Go core packages
│   ├── agent/            # Agent state, tool loops, context rollback
│   ├── claude/           # Claude Code CLI integration
│   ├── cli/              # Native CLI prompt loop & signal handlers
│   ├── config/           # Project & global configuration settings
│   ├── filesystem/       # Clean file operations & formatting
│   ├── git/              # Workspace git state inspection
│   └── ollama/           # Local LLM API streaming client
└── spec.md               # Technical specification
```

---

## 📜 License

Distributed under the [MIT License](LICENSE).

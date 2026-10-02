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

## 🚀 Installation & Usage

### 🍏 macOS Installation (Homebrew & Pre-compiled Binaries)

#### Option 1: Homebrew (Recommended for macOS)
```bash
brew tap NichuSPN/bandit
brew install bandit
```

#### Option 2: Build from Source on macOS
Prerequisites: Go `1.24+` and Rust (`cargo`).

```bash
git clone https://github.com/NichuSPN/bandit.git
cd bandit
./build.sh
```
This compiles `./bandit` and installs it to `~/.cargo/bin/bandit`.

---

### 🪟 Windows & Other OS (Run from Local Repository)

For Windows and other operating systems, clone the repository and run locally from source:

#### Prerequisites
- **Go**: Version `1.24+` ([golang.org](https://go.dev/dl/))
- **Rust**: Rust toolchain ([rustup.rs](https://rustup.rs/))
- **MinGW GCC**: GCC compiler for Windows CGO ([mingw-w64](https://www.mingw-w64.org/))

#### Running Locally on Windows
Open PowerShell or Command Prompt:

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
# Compile Rust engine static library
cd bandit_engine
cargo build --release
cd ..

# Run Go CLI
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

## 📦 How to Publish a New Release (For Maintainers)

Releases are automated via GitHub Actions (`.github/workflows/release.yml`).

To publish a new version:

```bash
git tag v1.0.0
git push origin v1.0.0
```

GitHub Actions will automatically build binaries for macOS (ARM64 & Intel) and Linux and publish them to GitHub Releases.

---

## 📜 License

Distributed under the [MIT License](LICENSE).

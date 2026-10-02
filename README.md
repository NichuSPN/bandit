# 🏴‍☠️ Bandit Terminal AI Agent

[![CI & Tests](https://github.com/NichuSPN/bandit/actions/workflows/ci.yml/badge.svg)](https://github.com/NichuSPN/bandit/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Bandit** is a high-performance, autonomous local terminal AI coding agent built with a hybrid Go CLI and CGO Rust execution engine. It interfaces locally with Ollama models and seamlessly escalates complex tasks to the interactive Claude Code CLI.

---

## 🔥 Key Features

- **Strict Read-Only Chat Boundary**: Standard interactive chat is strictly read-only—allowing you to inspect codebase diffs, search files, and plan changes without modifying disk state.
- **Controlled Disk Mutations (`/local` & `/claude`)**: Disk modifications occur strictly when explicitly triggered via `/local` or `/claude`.
- **Local Model Escalation Synthesis**: Running `/claude` invokes your local LLM to inspect session history, line numbers, and findings, synthesizing an optimized prompt before launching Claude CLI.
- **Multi-Line Paste Mode (`/multiline`)**: Dedicated `/paste` or `/multiline` mode (finish with `/end`) allows seamless entry of multi-line prompts and code blocks.
- **Custom Per-User & Project System Prompts**: Customize behavior per project (`./.bandit/system_prompt.txt`), globally (`~/.config/bandit/system_prompt.txt`), or via `config.json`.
- **Clean Formatting & Line Deletions**: Automatically collapses extra blank lines and cleanly trims indentation when deleting lines.
- **Domain Skills Auto-Discovery**: Automatically discovers skills across 4 locations (`./.bandit/skills`, `./.claude/skills`, `~/.bandit/skills`, `~/.claude/skills`).
- **Graceful Signal Interrupts (`Ctrl+C`)**: Interrupting an active step immediately cancels context and rolls back un-answered prompts from conversation history.

---

## 🚀 Quick Start & Installation

### macOS Installation

#### Option 1: Build from Source (Recommended)
Prerequisites: Go `1.24+` and Rust `cargo`.

```bash
git clone https://github.com/NichuSPN/bandit.git
cd bandit
./build.sh
```
This builds the single executable `./bandit` and installs it to `~/.cargo/bin/bandit`.

#### Option 2: Homebrew
```bash
brew tap NichuSPN/bandit
brew install bandit
```

---

### Windows Installation

#### Option 1: PowerShell Quick Installer
Open PowerShell and run:

```powershell
irm https://raw.githubusercontent.com/NichuSPN/bandit/main/install.ps1 | iex
```

#### Option 2: Build from Source
Prerequisites: Go `1.24+`, Rust `cargo`, and GCC/MinGW (for CGO).

```powershell
git clone https://github.com/NichuSPN/bandit.git
cd bandit
.\build.ps1
```

#### Option 3: Scoop Package Manager
```powershell
scoop bucket add bandit https://github.com/NichuSPN/bandit
scoop install bandit
```

---

### Direct Binary Download

Pre-compiled binary releases for **macOS (ARM64 & Intel)**, **Windows (x64)**, and **Linux (x64)** are published automatically on every release:

👉 **[Download Latest Binaries from GitHub Releases](https://github.com/NichuSPN/bandit/releases)**

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
├── install.ps1           # Windows installer script
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
└── packaging/            # Homebrew & Scoop packaging specs
    ├── homebrew/bandit.rb
    └── scoop/bandit.json
```

---

## 📦 How to Publish a New Release (For Maintainers)

Releases are automated via GitHub Actions (`.github/workflows/release.yml`).

To publish a new version:

```bash
git tag v1.0.0
git push origin v1.0.0
```

GitHub Actions will automatically build binaries for macOS, Windows, and Linux, compress them into release archives, and create a GitHub Release.

---

## 📜 License

Distributed under the [MIT License](LICENSE).

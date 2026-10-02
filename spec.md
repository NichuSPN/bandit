# Bandit

## 1. Overview

Bandit is a local-first terminal AI agent for developers.

Its primary purpose is to help the developer **understand and solve problems before spending premium-model tokens**.

The developer talks to Bandit using a terminal. Bandit uses a local coding-oriented LLM to:

- understand the user's problem
- inspect the filesystem
- inspect multiple repositories
- read relevant skills
- find related implementations and references
- discuss possible solutions with the user
- determine when a problem is large or complex enough to delegate to Claude
- prepare a concise, high-quality Claude prompt containing only the necessary context
- send the task to Claude when explicitly requested
- inspect Claude's resulting changes
- determine whether the task appears complete or requires follow-up
- tell the user what still needs to be verified

Bandit is **not an autonomous coding agent in V1**.

The user remains in control of decisions and execution.

---

# 2. Core Philosophy

Bandit follows this workflow:

~~~
User
  ↓
Bandit understands
  ↓
Bandit investigates
  ↓
Bandit discusses
  ↓
User decides
  ↓
Either:
    User implements the solution
or:
    Bandit prepares task for Claude
            ↓
          Claude
            ↓
      Bandit reviews changes
            ↓
          User
~~~

The key objective is:

> Use the local model for understanding, investigation, context gathering, and discussion. Use Claude only when the task benefits from Claude's stronger reasoning or when the implementation would be cumbersome to perform manually.

Bandit should avoid wasting Claude tokens on context gathering or simple questions.

---

# 3. Design Principles

## 3.1 Keep the implementation simple

Bandit is being written in Rust by a developer learning Rust.

The code must therefore prioritize:

- readability
- straightforward control flow
- small modules
- simple data structures
- explicit error handling
- minimal abstractions

Avoid unnecessary:

- frameworks
- complex trait hierarchies
- dependency injection systems
- generic abstractions
- macros
- async code unless required
- plugin architectures
- agent frameworks
- distributed components

Do not build abstractions for hypothetical future requirements.

Solve the current problem simply.

## 3.2 Local-first

Bandit should work primarily with a locally running model.

Claude is an optional escalation mechanism.

If Claude is unavailable or the user has exhausted their Claude allowance, Bandit should still be useful.

## 3.3 Read-only by default

Bandit should initially be a read-only agent.

It can inspect and reason about the user's environment without modifying it.

## 3.4 User-controlled execution

Bandit must never silently execute arbitrary commands.

Anything that changes or executes something in the user's environment requires explicit user approval.

---

# 4. Technology

## Language

Rust.

Use a modern stable Rust version.

Keep the code idiomatic but approachable to someone learning Rust.

## Local model

Use Ollama initially.

The model should be configurable.

Example:

~~~
qwen3-coder
~~~

The exact model name should not be hardcoded throughout the application.

Example configuration:

~~~toml
[model]
provider = "ollama"
model = "qwen3-coder"
base_url = "http://localhost:11434"
~~~

Bandit should communicate with Ollama over its HTTP API.

Do not implement local model inference inside Bandit.

## Claude

Use the Anthropic API.

The Claude API key should come from an environment variable.

Example:

~~~
ANTHROPIC_API_KEY
~~~

Do not store API keys in the Bandit configuration file.

---

# 5. CLI

Running:

~~~
bandit
~~~

starts an interactive terminal session.

Example:

~~~
$ bandit

Bandit
Local model: qwen3-coder
Workspace: ~/code

bandit>
~~~

The user can then have a normal conversation.

Example:

~~~
bandit> I'm getting duplicate events from my logger.
~~~

Bandit should respond conversationally while using tools when necessary.

---

# 6. CLI Commands

V1 should have a small number of commands.

## `/help`

Show available commands.

## `/clear`

Clear the current conversation.

## `/skills`

Show available skills.

## `/context`

Show the important context currently known to Bandit.

## `/claude`

Tell Bandit to prepare the current problem for Claude.

This should not automatically send anything.

## `/exit`

Exit Bandit.

Additional commands can be added later.

Do not build a large command system in V1.

---

# 7. Filesystem Access

Filesystem access is one of Bandit's core capabilities.

Bandit must not be restricted to the current repository.

The user will frequently refer to:

- another repository
- an old implementation
- reference projects
- documentation
- configuration files
- unrelated projects containing useful examples

Therefore Bandit needs access to configured filesystem roots.

Example:

~~~toml
[filesystem]
roots = [
    "~/code",
    "~/projects",
    "~/references"
]
~~~

Paths should support `~`.

Bandit may search within these configured roots.

The current working directory should also automatically be available.

---

# 8. Filesystem Tools

Bandit should expose simple tools to the local model.

## `list_directory`

List files and directories.

## `read_file`

Read a file.

The tool should support reasonable limits so that enormous files are not accidentally dumped into model context.

## `search_files`

Search filenames and/or file contents.

Use an existing search implementation such as `ripgrep` where appropriate.

Do not implement a custom search engine.

The local model should be able to ask:

~~~
Find occurrences of "OpenTelemetry" under ~/code.
~~~

and receive useful search results.

---

# 9. Git Support

Bandit should understand Git repositories.

For V1, Git operations may simply invoke the installed `git` command rather than using a Git Rust library.

Useful operations:

~~~
git status
git diff
git log
git branch
~~~

The primary operations are:

- inspect repository status
- inspect current diff
- inspect history when useful

Bandit must not automatically:

~~~
git commit
git push
git reset
git checkout
~~~

or otherwise modify repository state.

---

# 10. Skills

Skills are simple Markdown files containing persistent instructions and domain knowledge.

Example:

~~~
~/.config/bandit/skills/

    rust.md
    postgres.md
    react.md
    typescript.md
    opentelemetry.md
    data-nadhi.md
~~~

A skill may contain:

- project conventions
- technology-specific knowledge
- architectural rules
- preferred libraries
- things to avoid
- domain-specific context

Example:

~~~markdown
# PostgreSQL

The application uses PostgreSQL 16.

Prefer explicit SQL.

The application uses sqlx.

Avoid introducing an ORM unless specifically requested.
~~~

Skills are context available to Bandit throughout its operation.

---

# 11. Skill Discovery

Bandit should not inject every skill into every model request.

The local model should determine which skills are relevant.

For example:

~~~
User asks about PostgreSQL query
        ↓
Bandit identifies postgres.md
        ↓
Reads postgres.md
        ↓
Provides it as context
~~~

For a React problem:

~~~
react.md
typescript.md
possibly project-specific skill
~~~

The `/skills` command should display available skills.

---

# 12. Conversation

Bandit maintains conversation state during the current session.

The basic internal representation should remain simple.

Example:

~~~rust
struct Message {
    role: Role,
    content: String,
}

enum Role {
    System,
    User,
    Assistant,
    Tool,
}
~~~

Do not build persistent memory in V1.

Conversation history can exist for the duration of the session.

---

# 13. Agent Loop

The local model should be able to decide when it needs more context.

Basic flow:

~~~
User message
    ↓
Local model
    ↓
Does it need more information?
    │
    ├── No → respond
    │
    └── Yes
          ↓
        Tool
          ↓
      Tool result
          ↓
      Local model
          ↓
       response
~~~

Example:

~~~
User:
Why is authentication failing?

Bandit:
I need to inspect the authentication implementation.

→ searches filesystem

→ finds authentication middleware

→ reads relevant files

→ reads relevant skills

→ reasons about the problem

Bandit:
The likely problem is...
~~~

Do not implement a complex planner.

The local model can determine when to use tools.

---

# 14. Context Management

Bandit should avoid unnecessarily sending huge amounts of information to the local model or Claude.

The local model should retrieve context progressively.

Prefer:

~~~
search
↓
identify relevant files
↓
read relevant sections
~~~

rather than:

~~~
send entire repository
~~~

Similarly, Claude should receive a distilled context rather than the entire Bandit conversation.

---

# 15. Normal Problem-Solving Flow

A normal session should look like:

~~~text
User:
I'm getting duplicate events in the logger.

Bandit:
I'll investigate.

Bandit:
✓ Found relevant logging implementation
✓ Found collector implementation
✓ Found related implementation in another repository
✓ Read logging skill

Bandit:
The likely problem is that the same event is being
processed by both paths...

Here is the change I would make:
...

You can implement this yourself, or I can prepare it
for Claude.
~~~

If the user can solve it themselves, no Claude request is made.

This is the primary token-saving mechanism.

---

# 16. Claude Escalation

Claude should only be used when:

- the user explicitly requests it
- the task is sufficiently large
- the user does not want to implement the changes manually
- stronger reasoning is useful
- multiple files require coordinated changes

Bandit should never silently consume Claude tokens.

The user must explicitly approve sending a task to Claude.

---

# 17. Claude Prompt Generation

When escalating, Bandit should first create a structured task.

The task should contain:

~~~text
Problem

Context

Relevant files

Relevant code

Relevant skills

What has already been investigated

Required changes

Constraints

Acceptance criteria
~~~

Example:

~~~text
# Task

Fix duplicate logger events.

## Problem

The same event appears to be sent through both the
console and collector paths.

## Relevant context

The application has separate console and collector
handlers.

The current implementation...

## Relevant files

- src/logger.rs
- src/collector.rs
- src/events.rs

## Relevant skills

- opentelemetry.md
- data-nadhi.md

## Investigation

The duplicate appears to originate from...

## Required changes

1. ...
2. ...
3. ...

## Constraints

Do not change...
Do not introduce...

## Acceptance criteria

- ...
- ...
- ...
~~~

The generated prompt should be concise.

Do not include irrelevant conversation history.

---

# 18. Claude Approval

Before sending the request, Bandit should display the task.

Example:

~~~
╭─ Claude Task ─────────────────────────────────────╮
│ Fix duplicate logger events                       │
│                                                   │
│ 5 files                                           │
│ 2 relevant skills                                 │
│                                                   │
│ The task will be sent to Claude.                  │
╰───────────────────────────────────────────────────╯

Send to Claude? [Y/n]
~~~

Only after confirmation should Bandit call Claude.

---

# 19. Claude Response

Claude's response should not simply be printed and treated as final.

Bandit must process the response.

The response can contain:

- explanation
- implementation summary
- changed files
- remaining work
- warnings
- suggested tests
- limitations

Bandit should interpret the response and then inspect the actual repository.

---

# 20. Post-Claude Verification

After Claude responds, Bandit should inspect:

~~~text
git status
git diff
relevant changed files
~~~

It should compare the actual changes against the original task.

Bandit should determine one of:

~~~rust
enum TaskStatus {
    Complete,
    NeedsFollowUp,
    Blocked,
}
~~~

This is an assessment based on available information.

Bandit must not claim that runtime behaviour is correct merely because Claude said it was.

---

# 21. Post-Claude Review Example

Example:

~~~
╭─ Bandit · Review ─────────────────────────────────╮
│                                                   │
│ Claude changed 5 files.                           │
│                                                   │
│ ✓ Requested middleware added                      │
│ ✓ Error handling updated                           │
│ ✓ Tests added                                     │
│ ✓ No obvious incomplete implementation found      │
│                                                   │
│ Runtime tests have not been executed.             │
│                                                   │
│ Recommended next step:                            │
│ Run the project's test suite.                     │
╰───────────────────────────────────────────────────╯
~~~

Bandit should clearly distinguish:

- what it verified by inspecting files
- what Claude claimed
- what has not been verified

---

# 22. Tests and Command Execution

Bandit must **not automatically run tests**.

Bandit must not automatically run:

- test suites
- builds
- package managers
- application servers
- migrations
- arbitrary shell commands

After Claude makes changes, Bandit should tell the user what should be tested.

Example:

~~~
I have not run the tests.

Recommended:

    cargo test

Would you like Bandit to run this command?
~~~

If the user says no:

~~~
Run the test suite manually when you're ready.
~~~

If the user says yes:

1. Bandit displays the exact command.
2. Bandit asks for confirmation.
3. Only then executes it.
4. Reports the actual result.

Bandit must never claim tests passed unless they were actually executed and the result was observed.

---

# 23. Command Execution Policy

Bandit is read-only by default.

Automatically allowed:

~~~text
read files
search files
list directories
git status
git diff
git log
~~~

Not automatically allowed:

~~~text
write files
delete files
git commit
git push
git reset
run tests
run builds
install dependencies
start services
execute arbitrary shell commands
~~~

Commands outside the read-only set require explicit user approval.

---

# 24. Terminal UI

The interface should feel polished and sophisticated, similar in quality to modern AI coding tools.

It should remain a terminal application.

Do not build a GUI.

Useful visual elements:

- clear headers
- sections
- status indicators
- progress indicators
- tool activity
- highlighted code
- clear separation between user, Bandit, and Claude
- concise summaries
- confirmation prompts

Example:

~~~
╭─ Bandit ──────────────────────────────────────────╮
│ Local agent · qwen3-coder                          │
│ Workspace · ~/code/data-nadhi                      │
╰───────────────────────────────────────────────────╯

You
> Find where this event is being duplicated.

Bandit
  ◌ Searching workspace...
  ✓ 4 relevant files found
  ✓ Read opentelemetry skill
  ✓ Found related implementation

  I believe the duplication occurs because...

  ┌─ Suggested solution ────────────────────────────┐
  │ Separate event presentation from event          │
  │ collection at the logging boundary.             │
  └─────────────────────────────────────────────────┘
~~~

The UI should not become a separate architectural subsystem.

Keep presentation code simple.

---

# 25. Claude Interaction UI

Example:

~~~
Bandit
  ✓ Problem understood
  ✓ Relevant context gathered
  ✓ 6 files selected
  ✓ 2 skills included

  Claude task ready.

  ┌─ Task ──────────────────────────────────────────┐
  │ Implement the logger separation described above │
  └─────────────────────────────────────────────────┘

  Send to Claude? [Y/n]
~~~

During the Claude request:

~~~
Bandit
  ◌ Sending task to Claude...
  ◌ Waiting for response...
  ✓ Claude responded
  ◌ Reviewing changes...
  ✓ Review complete
~~~

---

# 26. Configuration

Configuration should live at:

~~~
~/.config/bandit/config.toml
~~~

Example:

~~~toml
[model]
provider = "ollama"
model = "qwen3-coder"
base_url = "http://localhost:11434"

[filesystem]
roots = [
    "~/code",
    "~/projects",
    "~/references"
]

[skills]
directory = "~/.config/bandit/skills"

[claude]
model = "..."
~~~

API credentials must be supplied through environment variables.

---

# 27. Suggested Rust Project Structure

Keep the project deliberately small.

~~~text
bandit/
├── Cargo.toml
├── src/
│   ├── main.rs
│   ├── cli.rs
│   ├── agent.rs
│   ├── model.rs
│   ├── ollama.rs
│   ├── claude.rs
│   ├── filesystem.rs
│   ├── git.rs
│   ├── skills.rs
│   ├── context.rs
│   └── error.rs
└── tests/
~~~

Responsibilities:

### `main.rs`

Application entry point.

### `cli.rs`

Interactive terminal loop and commands.

### `agent.rs`

Main conversation and tool-use loop.

### `model.rs`

Simple model interface and message structures.

### `ollama.rs`

Ollama implementation.

### `claude.rs`

Anthropic API integration.

### `filesystem.rs`

Filesystem operations.

### `git.rs`

Git inspection.

### `skills.rs`

Skill discovery and loading.

### `context.rs`

Context selection and Claude prompt generation.

### `error.rs`

Application error types.

Do not create more modules unless there is a concrete reason.

---

# 28. Dependencies

Prefer a small dependency set.

Reasonable choices include:

- `clap` for CLI arguments if needed
- `serde` / `serde_json` for API data
- `toml` for configuration
- `reqwest` for HTTP
- `tokio` only if required by the HTTP implementation
- `anyhow` or a simple custom error type
- a terminal UI library such as `ratatui` only if it genuinely simplifies the UI
- a color/terminal formatting library where useful

Do not add dependencies simply because another AI framework uses them.

---

# 29. No Persistent Memory in V1

Bandit does not need a database.

Do not implement:

- embeddings
- vector search
- semantic memory
- conversation database
- Redis
- SQLite

The configured filesystem and skill files provide persistent context.

Session history exists only for the current session.

Persistent memory can be added later if there is a demonstrated need.

---

# 30. No Autonomous Coding in V1

Bandit should not modify code automatically.

The intended workflow is:

~~~
Bandit → understands
Bandit → investigates
Bandit → recommends
User → implements

or

Bandit → understands
Bandit → investigates
Bandit → prepares Claude task
User → approves
Claude → implements
Bandit → reviews
User → decides next step
~~~

This keeps the system predictable.

---

# 31. Error Handling

Errors should be understandable.

Examples:

~~~
Could not connect to Ollama.

Expected Ollama at:
http://localhost:11434

Make sure Ollama is running and try again.
~~~

or:

~~~
Claude request failed.

Reason:
API authentication failed.

Check ANTHROPIC_API_KEY.
~~~

Do not expose raw Rust stack traces during normal CLI operation.

Detailed errors may be available through a debug option later.

---

# 32. Security

Filesystem access must be restricted to configured roots.

Bandit should reject attempts to access files outside those roots.

Examples:

~~~text
../../../etc/passwd
```

must not bypass the configured filesystem boundaries.

Sensitive files should not automatically be sent to Claude.

Bandit should make the user aware of what files are being included in an escalation.

---

# 33. Claude Context Privacy

Before sending a task to Claude, Bandit should show:

- relevant files
- relevant context
- approximate number of files
- relevant skills

The user should be able to inspect the generated Claude prompt.

The user explicitly approves the request before it is sent.

---

# 34. Completion Assessment

Bandit's assessment after Claude should be conservative.

If Claude says:

~~~
Implemented everything successfully.
~~~

Bandit should still inspect the diff.

If the original task required:

~~~text
1. API change
2. database change
3. frontend change
4. tests
~~~

Bandit should check whether those changes appear in the diff.

It can say:

~~~
✓ API implementation found
✓ Database migration found
✓ Frontend implementation found
✓ Tests added

Runtime behaviour has not been verified.
~~~

It must not claim more than it can establish.

---

# 35. Follow-Up Loop

If Bandit finds incomplete work:

~~~text
Claude response
      ↓
Bandit review
      ↓
Needs follow-up
      ↓
Explain remaining work
      ↓
Ask user
      ↓
Send follow-up to Claude if approved
      ↓
Review again
~~~

Example:

~~~
Bandit · Review

Claude completed most of the requested work.

✓ Backend implementation
✓ Database changes
✓ Frontend changes

⚠ Remaining:
The frontend still references the old API field.

I can prepare a follow-up task for Claude.

Prepare follow-up? [Y/n]
~~~

This loop can continue until:

~~~text
Complete
```

or the user ends the task.

---

# 36. Definition of Done for V1

Bandit V1 is complete when the following workflow works reliably:

~~~text
1. Start Bandit from terminal.

2. Talk naturally to the local model.

3. Bandit can inspect the current repository.

4. Bandit can inspect other repositories under configured
   filesystem roots.

5. Bandit can search and read files.

6. Bandit can inspect Git status and diff.

7. Bandit can discover and read skills.

8. Bandit can maintain conversation context.

9. Bandit can explain a solution without using Claude.

10. User can explicitly request Claude escalation.

11. Bandit creates a concise Claude task containing
    relevant context.

12. User reviews and approves the task.

13. Bandit sends the task to Claude.

14. Claude's response is returned to Bandit.

15. Bandit inspects git status and git diff after Claude.

16. Bandit determines whether the task appears complete
    or requires follow-up.

17. Bandit clearly identifies what has and has not been
    verified.

18. Bandit recommends tests without automatically running them.

19. Bandit asks permission before executing tests or
    other non-read-only commands.

20. The terminal UI feels polished and professional.
~~~

---

# 37. Explicit Non-Goals for V1

Do not implement:

- autonomous code modification by Bandit
- automatic test execution
- arbitrary shell execution
- MCP
- vector databases
- embeddings
- persistent memory
- multi-agent systems
- model routing between many providers
- GUI
- web application
- background daemon
- cloud-hosted Bandit
- automatic Claude escalation
- automatic Claude token management
- automatic commits
- automatic pushes
- automatic dependency installation

These can be considered later based on actual usage.

---

# 38. Guiding Principle

The most important requirement for the implementation is:

> **Bandit should feel sophisticated to use while remaining simple to understand internally.**

A developer learning Rust should be able to open any Bandit source file and understand what it does.

Prefer straightforward code over clever code.

The product should feel intelligent because the **local model, tools, context gathering, and interaction design** are good—not because the Rust architecture is complicated.

The final mental model should remain:

~~~
Bandit
  ├── talks to local model
  ├── reads files
  ├── reads skills
  ├── inspects git
  ├── prepares Claude tasks
  └── reviews Claude's work
~~~

That is Bandit V1.
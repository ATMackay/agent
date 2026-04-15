# Agent CLI — Architecture & Developer Documentation

> **Module:** `github.com/ATMackay/agent`
> **Language:** Go 1.25+
> **License:** MIT
> **Author:** Alex Mackay, 2026

---

## Table of Contents

1. [Overview](#overview)
2. [Project Structure](#project-structure)
3. [Architecture](#architecture)
4. [Packages](#packages)
   - [main](#main)
   - [cmd](#cmd)
   - [constants](#constants)
   - [model](#model)
   - [agents/documentor](#agentsdocumentor)
   - [agents/analyzer](#agentsanalyzer)
   - [tools](#tools)
   - [workflow](#workflow)
   - [state](#state)
5. [Agent Workflows](#agent-workflows)
   - [Documentor Agent](#documentor-agent)
   - [Analyzer Agent](#analyzer-agent)
6. [Tool System](#tool-system)
7. [LLM Provider Support](#llm-provider-support)
8. [Session & State Management](#session--state-management)
9. [Configuration & Environment Variables](#configuration--environment-variables)
10. [Build System](#build-system)
11. [CI Pipeline](#ci-pipeline)
12. [Extending the System](#extending-the-system)

---

## Overview

Agent CLI is a command-line application that runs AI agents powered by large language models (LLMs). It uses [Google's Agent Development Kit (ADK)](https://google.github.io/adk-docs/get-started/go/) as its agent runtime and supports multiple LLM providers (Anthropic Claude and Google Gemini).

The project currently ships two agent types:

| Agent | Purpose |
|---|---|
| **Documentor** | Fetches a GitHub repository, analyzes its code, and produces markdown documentation |
| **Analyzer** | Performs general-purpose filesystem and command-line tasks with a focus on document analysis |

---

## Project Structure

```
.
├── main.go                       # Entry point
├── Makefile                      # Build, install, run, and test targets
├── go.mod / go.sum               # Go module definition
├── LICENSE                       # MIT License
├── README.md                     # Quick-start guide
├── ARCHITECTURE.md               # This file
│
├── cmd/                          # CLI command definitions (cobra)
│   ├── cmd.go                    #   Root command
│   ├── run.go                    #   `run` parent command
│   ├── documentor.go             #   `run documentor` subcommand
│   ├── analyze.go                #   `run analyzer` subcommand
│   ├── version.go                #   `version` subcommand
│   ├── constants.go              #   CLI-scoped constants
│   └── logging.go                #   Structured logging setup with ANSI color
│
├── constants/                    # Build-time version metadata
│   ├── constants.go              #   Service name
│   └── version.go                #   Version, commit, build date (ldflags)
│
├── model/                        # LLM provider abstraction
│   ├── model.go                  #   Provider enum, Config, factory function
│   ├── claude.go                 #   Anthropic Claude provider
│   ├── gemini.go                 #   Google Gemini provider
│   └── model_test.go             #   Unit tests for model creation
│
├── agents/                       # Agent implementations
│   ├── documentor/
│   │   ├── documentor.go         #   Documentor agent construction
│   │   ├── prompt.go             #   System prompt & user message
│   │   └── config.go             #   Documentor configuration
│   └── analyzer/
│       ├── analyzer.go           #   Analyzer agent construction
│       ├── prompt.go             #   System prompt & user message
│       └── config.go             #   Analyzer configuration
│
├── tools/                        # Agent tool implementations
│   ├── tools.go                  #   Tool registry (Kind enum, GetTools)
│   ├── config.go                 #   Tool dependency injection (Deps)
│   ├── git_repo.go               #   fetch_repo_tree tool
│   ├── read_file.go              #   read_repo_file tool
│   ├── read_local_file.go        #   read_local_file tool
│   ├── write_file.go             #   write_output_file tool
│   ├── edit_file.go              #   edit_file tool
│   ├── exec_command.go           #   exec_command tool
│   ├── search_files.go           #   search_files tool
│   ├── list_dir.go               #   list_dir tool + path helpers
│   ├── tools_test.go             #   Unit tests for tools
│   └── fake_ctx_test.go          #   Test helper: fake tool.Context
│
├── workflow/                     # Agent execution runtime
│   ├── workflow.go               #   Workflow orchestrator
│   └── runner.go                 #   Runner interface
│
├── state/                        # Session state key constants
│   └── state.go
│
└── .github/workflows/
    └── go.yml                    # CI: build, test, lint
```

---

## Architecture

The system follows a layered architecture:

```
┌───────────────────────────────────────────┐
│                CLI (cmd)                  │  ← cobra commands, flag parsing, logging
├───────────────────────────────────────────┤
│              Workflow Engine              │  ← session creation, event loop
├───────────────────────────────────────────┤
│          Agent (agents/*)                 │  ← LLM agent with system prompt & tools
├──────────────────┬────────────────────────┤
│   Model (model)  │    Tools (tools)       │  ← LLM provider    ← callable functions
└──────────────────┴────────────────────────┘
│            Google ADK Runtime             │  ← runner, session, function-tool infra
└───────────────────────────────────────────┘
```

**Request flow:**

1. User invokes a CLI command (e.g. `agent run documentor --repo <url>`).
2. The command handler creates an LLM model via the `model` package.
3. An agent is constructed with its system prompt and tool set.
4. A `Workflow` is created, wrapping a Google ADK `runner.Runner` and an in-memory `session.Service`.
5. `Workflow.Start()` creates a session with initial state, sends the user message, and iterates over agent events (tool calls, LLM responses) until completion.
6. The agent calls tools (e.g. `fetch_repo_tree`, `read_repo_file`, `write_output_file`) which read/write files and update session state.
7. Final output is written to disk and the workflow terminates.

---

## Packages

### `main`

**File:** `main.go`

The entry point. Creates the root cobra command via `cmd.NewAgentCLICmd()` and executes it. Exits with code 1 on error.

---

### `cmd`

The CLI layer built on [cobra](https://github.com/spf13/cobra) and [viper](https://github.com/spf13/viper).

| File | Responsibility |
|---|---|
| `cmd.go` | Root command `agent [subcommand]`. Registers `run` and `version`. |
| `run.go` | Parent `run` command. Initializes logging, prints version info, registers agent subcommands (`documentor`, `analyzer`). |
| `documentor.go` | `run documentor` — configures and executes the documentor agent workflow. |
| `analyze.go` | `run analyzer` — configures and executes the analyzer agent workflow. |
| `version.go` | `version` — prints build metadata (semver, git commit, build date, dirty flag). |
| `logging.go` | Configures `slog` with text or JSON handlers. Includes ANSI color-coded log levels for terminal output. |
| `constants.go` | Defines `userCLI` constant used as the ADK session user ID. |

**Environment variable prefix:** `AGENT` (e.g. `AGENT_LOG_LEVEL`).

**API key resolution order:** `--api-key` flag → `API_KEY` → `GOOGLE_API_KEY` → `GEMINI_API_KEY` → `CLAUDE_API_KEY`.

---

### `constants`

Build-time metadata injected via Go linker flags (`-ldflags`).

| Variable | Description |
|---|---|
| `ServiceName` | `"agent-cli"` |
| `Version` | Semantic version from `git describe --tags` |
| `GitCommit` | Full commit SHA |
| `CommitDate` | Commit timestamp (UTC) |
| `BuildDate` | Binary build timestamp (UTC) |
| `Dirty` | `"true"` if the working tree has uncommitted changes |

---

### `model`

Abstracts LLM provider selection behind the Google ADK `model.LLM` interface.

**Supported providers:**

| Provider | Constant | Default Model | SDK |
|---|---|---|---|
| **Claude** (default) | `ProviderClaude` | `claude-opus-4-1-20250805` | `anthropic-sdk-go` via `claude-go-adk` adapter |
| **Gemini** | `ProviderGemini` | `gemini-2.5-pro` | `google.golang.org/genai` via ADK's built-in Gemini model |

**Key types:**

- `Provider` — string enum (`"claude"`, `"gemini"`).
- `Config` — provider name, model name, and (private) API key.
- `New(ctx, cfg) (model.LLM, error)` — factory function; defaults to Claude when provider is empty.

---

### `agents/documentor`

**Purpose:** Fetch a GitHub repository, read its source files, and generate markdown documentation.

**Construction:** `NewDocumentor(ctx, cfg, model) (*Documentor, error)`

**Tools used:**
- `fetch_repo_tree` — Clone/download the repository and build a file manifest.
- `read_repo_file` — Read files from the cached checkout with line-range support.
- `search_files` — Search for text patterns across the repository.
- `write_output_file` — Write the final markdown documentation to disk.

**System prompt highlights:**
- Instructs the agent to call `fetch_repo_tree` first, then selectively read files.
- Prioritizes search-before-read to minimize token usage.
- Enforces a `max_files` limit.
- Targets entry points, core packages, config, interfaces, and constructors.
- Explicitly avoids tests, mocks, fixtures, vendor, and generated files.
- Outputs structured markdown documentation stored in session state under the `documentation_markdown` key.

---

### `agents/analyzer`

**Purpose:** General-purpose agent for filesystem exploration, document analysis, file editing, and shell commands.

**Construction:** `NewAnalyzer(ctx, cfg, model) (*Analyzer, error)`

**Tools used:**
- `list_dir` — Explore directory trees.
- `read_local_file` — Read text files from the local filesystem.
- `write_output_file` — Write output files.
- `edit_file` — Targeted string replacement in files.
- `exec_command` — Run shell commands (build, extract text from PDFs, etc.).
- `search_files` — Search for text patterns across local files.

**System prompt highlights:**
- Instructs the agent to understand the task, explore with `list_dir`, search before reading, and use snippet reads.
- Provides guidance for binary document analysis (PDF via `pdftotext`, DOCX via `pandoc`, archives via `unzip`).
- Emphasizes efficiency: explore → search → snippet-read → act → write output.

---

### `tools`

The tool system provides typed, ADK-compatible function tools. Each tool is a closure that captures dependencies and returns structured results.

**Registry pattern:**

```go
// Kind is a string enum identifying each tool
type Kind string

// GetTools resolves a list of Kinds into initialized tool.Tool values
func GetTools(kinds []Kind, deps *Deps) ([]tool.Tool, error)
```

**Dependency injection:**

Tools that need configuration (e.g. `fetch_repo_tree` needs a `WorkDir`) receive it through `Deps`, a type-safe config map:

```go
deps := tools.Deps{}
deps.AddConfig(tools.FetchRepoTree, tools.FetchRepoTreeConfig{WorkDir: workDir})
```

**Available tools:**

| Tool Name | Kind Constant | Description |
|---|---|---|
| `fetch_repo_tree` | `FetchRepoTree` | Downloads a GitHub repository (HTTPS tarball or `git clone` fallback), extracts it, builds a source-file manifest, and stores the local path in session state. |
| `read_repo_file` | `ReadFile` | Reads a file from the cached repository checkout. Supports line ranges (`start_line`/`end_line`), byte limits, and full-file reads. Tracks loaded files in state. |
| `read_local_file` | `ReadLocalFile` | Reads a file from the local filesystem relative to `work_dir`. Same snippet capabilities as `read_repo_file`. |
| `write_output_file` | `WriteFile` | Writes content to a file path (from args or session state). Creates parent directories. Stores content in session state. |
| `edit_file` | `EditFile` | Exact string replacement in local files. Single-match by default (errors on ambiguity); `replace_all` flag available. |
| `exec_command` | `ExecCommand` | Executes a shell command with configurable timeout (default 30s, max 300s). Returns stdout, stderr, exit code, and timeout flag. Output capped at 64 KB per stream. |
| `search_files` | `SearchFiles` | Case-insensitive text search across files in `work_dir` or cached repo. Returns matching paths, line numbers, and context snippets. Max 100 results, 0–3 context lines. |
| `list_dir` | `ListDir` | Lists directory contents up to a configurable depth (default 3, max 10). Returns paths, kinds (`file`/`dir`), and sizes. Skips common non-source directories. |

**Security considerations:**
- `fetch_repo_tree` validates archive entries against path traversal (zip-slip protection via `isWithinBase`).
- `read_repo_file` validates that requested paths don't escape the repository root.
- `exec_command` uses bounded timeouts and output limits.
- `edit_file` requires an exact `old_string` match, preventing accidental bulk changes without `replace_all`.
- Symlinks are ignored in archive extraction and manifest building.

---

### `workflow`

Orchestrates agent execution using the Google ADK runner and session infrastructure.

**Key types:**

- `Runner` — interface wrapping the ADK `runner.Runner.Run()` method (returns `iter.Seq2[*session.Event, error]`).
- `Workflow` — holds a runner, session service, and initial state.

**`Workflow.Start(ctx, userID, userMsg)`:**

1. Creates a new ADK session with initial state.
2. Calls `runner.Run()` which starts the LLM agent loop.
3. Iterates over events, logging token usage metadata (total, prompt, tool-use, thought tokens).
4. Logs response content at `DEBUG` level.
5. Reports total execution time.

---

### `state`

Defines session state key constants shared across agents and tools.

| Key | Used By | Description |
|---|---|---|
| `output_path` | Both | Path for the output file |
| `repo_url` | Documentor | Repository URL |
| `repo_ref` | Documentor | Git ref (branch/tag/commit) |
| `sub_path` | Documentor | Subdirectory filter |
| `max_files` | Documentor | Maximum files to read |
| `temp_repo_manifest` | Documentor | JSON file manifest |
| `temp_repo_local_path` | Documentor | Local checkout path |
| `temp_loaded_files` | Documentor | Tracking of files already read |
| `documentation_markdown` | Documentor | Final documentation content |
| `work_dir` | Analyzer | Working directory for local operations |

---

## Agent Workflows

### Documentor Agent

```
CLI input: --repo <url> [--ref <ref>] [--path <subdir>] [--output doc.agentcli.md]
                │
                ▼
        ┌───────────────┐
        │  fetch_repo_tree  │──→ Download repo → Build manifest → Store in state
        └───────┬───────┘
                │
                ▼
        ┌───────────────┐
        │  search_files    │──→ Locate relevant symbols/types/functions
        └───────┬───────┘
                │
                ▼
        ┌───────────────┐
        │  read_repo_file  │──→ Read targeted file snippets (≤ max_files)
        └───────┬───────┘
                │
                ▼
        ┌─────────────────┐
        │ write_output_file │──→ Write markdown to --output path
        └─────────────────┘
```

### Analyzer Agent

```
CLI input: --task "..." [--work-dir <dir>] [--output analysis.md]
                │
                ▼
        ┌──────────┐
        │  list_dir   │──→ Explore directory structure
        └────┬─────┘
             │
             ▼
        ┌──────────────┐
        │  search_files   │──→ Find relevant content
        └────┬─────────┘
             │
             ▼
     ┌───────────────┐
     │ read_local_file │──→ Read targeted snippets
     └───────┬───────┘
             │
             ▼
   ┌─────────────────┐
   │  exec_command     │──→ Run shell commands (optional)
   │  edit_file        │──→ Modify files (optional)
   └────────┬────────┘
             │
             ▼
   ┌─────────────────┐
   │ write_output_file │──→ Write result to --output path
   └─────────────────┘
```

---

## LLM Provider Support

Both agents support switching providers at runtime:

```bash
# Use Claude (default)
agent run documentor --repo <url> --provider claude --model claude-opus-4-1-20250805

# Use Gemini
agent run analyzer --task "..." --provider gemini --model gemini-2.5-pro
```

The `model` package implements a factory pattern: `model.New()` dispatches to the appropriate provider constructor based on the `Provider` field. Claude is the default when no provider is specified.

The Anthropic integration uses the community `claude-go-adk` adapter to bridge the Anthropic SDK with Google ADK's `model.LLM` interface.

---

## Session & State Management

Sessions are managed using Google ADK's `session.InMemoryService()`. Each agent run creates a fresh session with initial state derived from CLI flags.

State flows through the system in two ways:
1. **Initial state** — set by the CLI command from flags/args (e.g. `repo_url`, `work_dir`, `output_path`).
2. **State deltas** — tools update state via `ctx.Actions().StateDelta[key] = value` during execution.

The agent's system prompt uses template variables (e.g. `{repo_url}`, `{work_dir}`) that are automatically resolved from session state by the ADK runtime.

---

## Configuration & Environment Variables

| Flag | Env Var(s) | Default | Description |
|---|---|---|---|
| `--log-level` | `AGENT_LOG_LEVEL` | `info` | Log level: debug, info, warn, error |
| `--log-format` | `AGENT_LOG_FORMAT` | `text` | Log format: text (colored), json |
| `--api-key` | `API_KEY`, `GOOGLE_API_KEY`, `GEMINI_API_KEY`, `CLAUDE_API_KEY` | *(required)* | LLM API key |
| `--provider` | `AGENT_PROVIDER` | `claude` | LLM provider: claude, gemini |
| `--model` | `AGENT_MODEL` | `claude-opus-4-1-20250805` | LLM model name |
| `--repo` | `AGENT_REPO` | *(required for documentor)* | GitHub repository URL |
| `--ref` | `AGENT_REF` | HEAD | Git ref (branch/tag/commit) |
| `--path` | `AGENT_PATH` | *(root)* | Subdirectory to document |
| `--output` | `AGENT_OUTPUT` | `doc.agentcli.md` / `analysis.md` | Output file path |
| `--max-files` | `AGENT_MAX_FILES` | `50` | Max files to read (documentor) |
| `--task` | `AGENT_TASK` | *(required for analyzer)* | Task description (analyzer) |
| `--work-dir` | `AGENT_WORK_DIR` | current directory | Working directory (analyzer) |

---

## Build System

The `Makefile` provides the following targets:

| Target | Description |
|---|---|
| `make build` | Compile the binary to `build/agent-cli` with version metadata via ldflags |
| `make install` | Build and move binary to `$GOBIN` |
| `make run` | Build and run the documentor agent on the project's own repository |
| `make test` | Run all tests with coverage output to `build/coverage/ut_cov.out` |

**Build output:** `build/agent-cli`

**Version injection:** The Makefile derives version, commit SHA, commit date, build date, and dirty flag from git, then injects them into the `constants` package via `-ldflags -X`.

---

## CI Pipeline

**File:** `.github/workflows/go.yml`

| Job | Steps |
|---|---|
| `unit-test` | Checkout → Setup Go 1.26 → `go build ./...` → `go test -v -cover ./...` |
| `golangci` | Checkout → Setup Go 1.26 → Run `golangci-lint` with 2-minute timeout |

**Triggers:** Push to `main`, PRs targeting `main`.

---

## Extending the System

### Adding a new agent

1. Create a new package under `agents/<name>/` with:
   - `config.go` — configuration struct with `Validate()`.
   - `prompt.go` — system prompt (`buildInstruction()`) and `UserMessage()`.
   - `<name>.go` — constructor using `llmagent.New()` with selected tools.
2. Select tools from `tools.Kind` or implement new ones.
3. Add a new cobra subcommand under `cmd/` (register it in `run.go`).

### Adding a new tool

1. Define args/result structs in a new file under `tools/`.
2. Implement the tool function with signature `func(tool.Context, Args) (Result, error)`.
3. Wrap it with `functiontool.New()`.
4. Add a new `Kind` constant and register it in `GetToolByEnum()`.
5. Write unit tests using `newFakeToolContext()` from `fake_ctx_test.go`.

### Adding a new LLM provider

1. Add a new `Provider` constant in `model/model.go`.
2. Implement a constructor function (`func newProvider(ctx, cfg) (model.LLM, error)`).
3. Add the case to the `New()` switch statement.
4. The provider must satisfy Google ADK's `model.LLM` interface.

---

## Testing

Run the full test suite:

```bash
make test
# or
go test -v -cover ./...
```

The `tools` package includes unit tests for `edit_file`, `list_dir`, `read_local_file`, and `exec_command` using a `fakeToolContext` that provides an in-memory session state implementation. The `model` package tests provider construction with valid and invalid configurations.

---

*Generated from source analysis of the `feat/sequential-agent` branch.*

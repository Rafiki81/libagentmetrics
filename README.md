# libagentmetrics

[![Go Reference](https://pkg.go.dev/badge/github.com/Rafiki81/libagentmetrics.svg)](https://pkg.go.dev/github.com/Rafiki81/libagentmetrics)
[![Go Report Card](https://goreportcard.com/badge/github.com/Rafiki81/libagentmetrics)](https://goreportcard.com/report/github.com/Rafiki81/libagentmetrics)

A Go library for real-time detection, monitoring, and analysis of AI coding agents. Zero external dependencies — stdlib only.

## What's New in v0.2.0

- **Global monitor health** via `BuildHealthReport(...)` for quick operational visibility.
- **Budget guardrails** with daily/monthly limits and fleet-level checks (`CheckFleet`).
- **Better resilience** with safer monitor defaults (zero-value safe) and tighter fallback behavior.
- **Performance-focused guidance** in docs to keep overhead low in production loops.

## Features

- **Auto-detection** of 12 agents: Claude Code, GitHub Copilot, Cursor, Aider, Cody, Continue.dev, Windsurf, Gemini CLI, OpenAI Codex CLI, Open Codex, MoltBot, Codel.
- **Process metrics** — CPU, memory, open files per PID.
- **Tokens & cost** — Real log parsing (Copilot, Claude JSONL, Cursor SQLite, Aider) with network-based estimation fallback, plus per-metric confidence score. Per-model cost calculation.
- **Git activity** — Branch, recent commits, diff stats, lines of code.
- **Terminal** — Detection of commands spawned by agent child processes.
- **Session** — Active vs. idle time based on CPU usage.
- **Network** — Active connections via `lsof`.
- **Filesystem** — File change watcher using polling.
- **Security** — Detection of dangerous commands, privilege escalation, reverse shells, credential access, exfiltration, and more (18 categories).
- **Alerts** — Configurable thresholds for CPU, memory, tokens, cost and idle time.
- **Local models** — Detection of Ollama, LM Studio, vLLM, llama.cpp, LocalAI, text-generation-webui, GPT4All.
- **History** — Persistent recording with JSON and CSV export.

## Installation

```bash
go get github.com/Rafiki81/libagentmetrics
```

Requires **Go 1.24+**. No third-party dependencies.

## Project Structure

```
libagentmetrics/
├── agent/          # Types, agent registry and process detection
│   ├── types.go    # Info, Instance, Snapshot, TokenMetrics, SecurityEvent, ...
│   ├── registry.go # 12 pre-registered agents
│   └── detector.go # Process scanner
├── config/         # JSON configuration with defaults
│   └── config.go   # Config, AlertConfig, SecurityConfig, LocalModelsConfig, ...
├── monitor/        # Monitoring modules
│   ├── alerts.go       # AlertMonitor — thresholds and alert generation
│   ├── cost.go         # Per-model cost estimation (OpenAI, Anthropic, Google)
│   ├── filesystem.go   # FileWatcher — directory change polling
│   ├── git.go          # GitMonitor — branch, commits, diff, LOC
│   ├── history.go      # HistoryStore — persistent recording, JSON/CSV export
│   ├── localmodels.go  # LocalModelMonitor — Ollama, LM Studio, vLLM, etc.
│   ├── network.go      # NetworkMonitor — connections via lsof
│   ├── process.go      # ProcessMonitor — CPU/memory per PID
│   ├── security.go     # SecurityMonitor — 18 event categories
│   ├── session.go      # SessionMonitor — uptime, active/idle
│   ├── terminal.go     # TerminalMonitor — child process commands
│   └── tokens.go       # TokenMonitor — Copilot, Claude, Cursor, Aider, network
└── examples/
    └── basic/main.go   # Full working example
```

## Quick Start

```go
package main

import (
    "fmt"

    "github.com/Rafiki81/libagentmetrics/agent"
    "github.com/Rafiki81/libagentmetrics/config"
    "github.com/Rafiki81/libagentmetrics/monitor"
)

func main() {
    cfg := config.DefaultConfig()
    registry := agent.NewRegistry()
    detector := agent.NewDetector(registry, cfg)

    // Scan for running agents
    agents, err := detector.Scan()
    if err != nil {
        panic(err)
    }

    // Collect metrics
    sessMon := monitor.NewSessionMonitor()
    gitMon := monitor.NewGitMonitor()
    tokenMon := monitor.NewTokenMonitor()

    for i := range agents {
        a := &agents[i]
        sessMon.Collect(a)
        gitMon.Collect(a)
    }
    tokenMon.Collect(agents)

    // Print results
    for _, a := range agents {
        fmt.Printf("%s (PID %d) — CPU: %.1f%% — Tokens: %s\n",
            a.Info.Name, a.PID, a.CPU,
            monitor.FormatTokenCount(a.Tokens.TotalTokens))
    }
}
```

## Packages

### `agent`

| Type | Description |
|------|-------------|
| `Info` | Metadata for a known agent (name, ID, process patterns) |
| `Instance` | A running instance with all its collected metrics |
| `Snapshot` | Point-in-time capture of all agents and alerts |
| `Registry` | Registry of the 12 supported agents |
| `Detector` | Process scanner that returns `[]Instance` |

### `config`

| Function | Description |
|----------|-------------|
| `DefaultConfig()` | Returns configuration with sensible defaults |
| `Load(path)` | Loads configuration from a JSON file |
| `Save(path)` | Saves configuration to a JSON file |
| `ConfigPath()` | Default path: `~/.config/agentmetrics/config.json` |

### `monitor`

| Monitor | Constructor | Description |
|---------|------------|-------------|
| `ProcessMonitor` | `NewProcessMonitor(pids)` | CPU, memory, open files per PID |
| `SessionMonitor` | `NewSessionMonitor()` | Uptime, active/idle time |
| `TerminalMonitor` | `NewTerminalMonitor(maxHistory)` | Commands spawned by the agent |
| `TokenMonitor` | `NewTokenMonitor()` | Tokens from logs, DB or network |
| `GitMonitor` | `NewGitMonitor()` | Branch, commits, diff stats |
| `NetworkMonitor` | `NewNetworkMonitor()` | Active network connections |
| `FileWatcher` | `NewFileWatcher()` | Directory change polling |
| `AlertMonitor` | `NewAlertMonitor(thresholds)` | Threshold-based alerts |
| `SecurityMonitor` | `NewSecurityMonitor(cfg)` | Suspicious activity detection |
| `LocalModelMonitor` | `NewLocalModelMonitor(cfg)` | Local models (Ollama, etc.) |
| `HistoryStore` | `NewHistoryStore()` | Persistent recording with export |

#### Formatting Helpers

```go
monitor.FormatTokenCount(int64) string     // "1.5k", "2.3M"
monitor.FormatTokensPerSec(float64) string // "45/s", "1.2k/s"
monitor.FormatDuration(time.Duration) string // "1h 23m", "45s"
monitor.FormatCost(float64) string          // "$0.0234"
monitor.EstimateCost(model, in, out) float64
```

## Performance Budget

This library is designed to be lightweight, but some monitors invoke system commands.
To keep overhead predictable, use different sampling intervals by monitor cost:

- **Fast path (every 2-3s):** `SessionMonitor`, `AlertMonitor`, `TokenMonitor` (when logs are available).
- **Medium path (every 5-10s):** `ProcessMonitor`, `TerminalMonitor`, `NetworkMonitor`.
- **Slow path (every 15-30s):** `GitMonitor`, `SecurityMonitor`, `LocalModelMonitor`.

### Practical Recommendations

- Reuse monitor instances across cycles (avoid re-creating monitors in tight loops).
- Call `detector.Scan()` on a slower cadence than lightweight metric refresh if agent churn is low.
- Prefer constructors (`NewGitMonitor`, `NewNetworkMonitor`, etc.) for clearer intent and internal state reuse.
- For near-real-time UIs, run a split loop: fast loop for CPU/session/tokens and slow loop for git/network/security.
- If running on battery-constrained environments, increase slow-path interval first.

### Notes on Cost Drivers

- `TokenMonitor` may use `sqlite3`, `nettop`, and `lsof` fallbacks depending on available sources.
- `NetworkMonitor` and `ProcessMonitor` rely on `lsof`/`ps` and are usually the first knobs to tune for lower overhead.
- `GitMonitor` cost depends on repository size and uncommitted diff volume.

## Budget Configuration (Daily/Monthly)

You can enable fleet-level budget alerts (cost + token context) without adding extra monitor loops.

### `config.json` example

```json
{
    "alerts": {
        "daily_budget_usd": 15,
        "monthly_budget_usd": 300,
        "budget_warn_percent": 80,
        "burn_rate_warning": 2.0,
        "burn_rate_critical": 3.0
    }
}
```

### Usage

- Keep your per-agent checks as usual with `alertMon.Check(&agent)`.
- Add one aggregated call after token collection:

```go
tokenMon.Collect(agents)
alertMon.CheckFleet(agents)
```

When enabled, this produces warning/critical alerts for:

- Daily budget high usage / exceeded.
- Monthly budget high usage / exceeded.
- Daily/monthly burn-rate above expected pace.

Token metrics also expose `confidence` (`0.0-1.0`) based on source reliability (`log/db > estimated > network`).

## Supported Agents

| Agent | ID | Detection |
|-------|----|-----------|
| Claude Code | `claude-code` | Process + JSONL logs |
| GitHub Copilot | `copilot` | Process + VS Code logs |
| Cursor | `cursor` | Process + SQLite DB |
| Aider | `aider` | Process + markdown history |
| Cody (Sourcegraph) | `cody` | Process |
| Continue.dev | `continue` | Process |
| Windsurf | `windsurf` | Process |
| Gemini CLI | `gemini-cli` | Process |
| OpenAI Codex CLI | `openai-codex` | Process |
| Open Codex | `open-codex` | Process |
| MoltBot | `moltbot` | Process |
| Codel | `codel` | Process |

## Security

The `SecurityMonitor` evaluates 18 event categories across 4 severity levels:

**Categories:** dangerous commands, privilege escalation, code injection, system modification, package installation, reverse shell, obfuscation, container escape, environment variable manipulation, credential access, log tampering, remote access, shell persistence, sensitive files, network exfiltration, mass deletion, secrets exposure, suspicious network.

**Severities:** `LOW`, `MEDIUM`, `HIGH`, `CRITICAL`

## Platform

Designed for **macOS**. Uses system tools such as `ps`, `lsof`, `pgrep`, `nettop`, and `git`. Linux support is possible with minor adjustments to log paths and system commands.

## API Stability (v1)

Starting with `v1.x`, the project follows semantic versioning for public APIs:

- **No breaking changes in minor/patch versions** (`v1.y.z`).
- **Breaking changes only in a new major version** (`v2.0.0`, etc.).
- Public types/functions documented in this README and package docs are considered stable.
- Internal helpers and unexported symbols may change at any time.

## Commits and Changelog

For maintainability, this project prefers **Conventional Commits** in PR titles/merge commits:

- `feat: ...`, `fix: ...`, `docs: ...`, `perf: ...`, `refactor: ...`, `test: ...`, `ci: ...`, `chore: ...`

Release notes/changelog entries are generated automatically when creating a GitHub Release from a semantic tag (e.g. `v1.2.3`).

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for branch strategy, validation checklist, and recommended branch protection rules.

## License

MIT

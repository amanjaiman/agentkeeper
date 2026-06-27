# Changelog

All notable changes to AgentKeeper are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project aims to
follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Core loop (M1)** — launch a coding agent in a managed tmux session, detect the
  usage-limit message, parse the reset time, wait, and inject a static resume prompt.
- **Graceful detach (M2)** — `d`/`q`/`k` hotkeys, `detach`/`stop` commands, a
  per-instance state file with `status`, auto-detach when a human attaches, and
  Ctrl-C that detaches rather than kills.
- **Codex adapter (M3)** — a second agent driven by the same loop via config-driven
  patterns; relative-duration reset parsing (`in 2h30m`); `agents` command.
- **Local-LLM reprompt (M4)** — `--reprompt ollama:<model>` reads the transcript
  tail + `git diff`, asks a local model for a continuation instruction, validates it
  (length + denylist), and falls back to the static prompt on any failure.
- **Polish (M5)** — `--backend pty` no-tmux fallback (Unix), desktop + `--webhook`
  notifications, progressive re-limit backoff.
- **Operability** — `attach-existing` (watch/recover a running session),
  `--watch-only` (notify but don't inject), `--yolo` (explicit unattended opt-in),
  `parse` (test limit strings against patterns), and a `version` command.
- **Background mode** — `run --daemon` re-executes detached from the terminal, logs
  to `<state-dir>/<name>.log`, and is controlled entirely via `status` / `detach` /
  `stop`. Works with both backends: the tmux backend keeps full handoff; the pty
  backend (incl. Windows ConPTY) runs headless and ends on detach. Cross-platform
  detach (setsid on Unix, detached process group on Windows).
- **Native Windows support** — a ConPTY-based `pty` backend (the default on
  Windows) runs a native Windows agent in a pseudoconsole, so AgentKeeper works on
  Windows with no WSL, including `--daemon`. Linux/macOS/Windows are all
  first-class.
- **Dead-session detection** — the supervisor now stops cleanly (new `ENDED`
  state + notification) when the agent exits or the tmux session is killed out
  from under it, instead of looping forever on `capture failed`. Consecutive
  capture failures are bounded as a safety net for a persistently broken backend.

### Fixed
- The watch loop no longer spins indefinitely when the supervised session
  disappears; `run` exits within a couple of poll intervals.

[Unreleased]: https://github.com/amanjaiman/agentkeeper/commits/main

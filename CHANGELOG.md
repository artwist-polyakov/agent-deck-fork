# Changelog

All notable changes to Agent Deck will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.8.89] - 2026-01-28

### Fixed

- Fix shutdown hang when quitting with "shut down" MCP pool option (process `Wait()` blocked forever on child-held pipes)
- Set `cmd.Cancel` (SIGTERM) and `cmd.WaitDelay` (3s) on MCP processes for graceful shutdown with escalation
- Add 5s safety timeout to individual proxy `Stop()` and 10s overall timeout to pool `Shutdown()`

## [0.8.88] - 2026-01-28

### Fixed

- Fix stale expanded group state during reload causing cursor jumps when CLI adds a session while TUI is running
- Fix new groups added via CLI appearing collapsed instead of expanded
- Eliminate redundant tree rebuild and viewport sync during reload (performance)

## [0.8.87] - 2026-01-28

### Added

- Add `env` field to custom tool definitions for inline environment variables (closes #101)
- Custom tools from config.toml now appear in the TUI command picker with icons
- CLI `agent-deck add -c <custom-tool>` resolves tool to actual command automatically

### Fixed

- Fix `[worktree] default_location = "subdirectory"` config not being applied (fixes #110)
- Add `--location` CLI flag to override worktree placement per session (`sibling` or `subdirectory`)
- Worktree location now respects config in both CLI and TUI new session dialog

## [0.8.86] - 2026-01-28

### Fixed

- Fix changelog display dropping unrecognized lines (plain text paragraphs now preserved)
- Fix trailing-slash path completion returning directory name instead of listing contents
- Reset path autocomplete state when reopening new session dialog
- Fix double-close on LogWatcher and StorageWatcher (move watcher.Close inside sync.Once)
- Fix log worker shutdown race (replace unused channel with sync.WaitGroup)
- Fix CapturePane TOCTOU race with singleflight deduplication

### Added

- Comprehensive test suite for update package (CompareVersions, ParseChangelog, GetChangesBetweenVersions, FormatChangelogForDisplay)

## [0.8.85] - 2026-01-27

### Fixed

- Clear MCP cache before regeneration to prevent stale reads
- Cursor jump during navigation and view duplication bugs

## [0.8.83] - 2026-01-26

### Fixed

- Resume with empty session ID opens picker instead of random UUID
- Subgroup creation under selected group

### Added

- Fast text copy (`c`) and inter-session transfer (`x`)

## [0.8.79] - 2026-01-26

### Added

- Gemini model selection dialog (`Ctrl+G`)
- Configurable maintenance system with TUI feedback
- Improved status detection accuracy and Gemini prompt caching
- `.env` file sourcing support for sessions (`[shell] env_files`)
- Default dangerous mode for power users

### Fixed

- Sync session IDs to tmux env for cross-project search
- Write headers to Claude config for HTTP MCPs
- OpenCode session detection persistence and "Detecting session..." bug
- Preserve parent path when renaming subgroups

## [0.8.69] - 2026-01-20

### Added

- MCP Manager user scope: attach MCPs to `~/.claude.json` (affects all sessions)
- Three-scope MCP system: LOCAL, GLOBAL, USER
- Session sharing skill (export/import sessions between developers)
- Scrolling support for help overlay on small screens

### Fixed

- Prevent orphaned test sessions
- MCP pool quit confirmation

## [0.8.67] - 2026-01-20

### Added

- Notification bar enabled by default
- Thread-safe key bindings for background sync
- Background worker self-ticking for status updates during `tea.Exec`
- `ctrl+c to interrupt` as primary busy indicator detection
- Debug logging for status transitions

### Changed

- Reduced grace period from 5s to 1.5s for faster startup detection
- Removed 6-second animation minimum; uses status-based detection
- Hook-based polling replaces frequent tick-based detection

## [0.8.65] - 2026-01-19

### Improved

- Notification bar performance and active session detection
- Increased busy indicator check depth from 10 to 20 lines

## [0.6.1] - 2025-12-24

### Changed

- **Replaced Aider with OpenCode** - Full integration of OpenCode (open-source AI coding agent)
  - OpenCode replaces Aider as the default alternative to Claude Code
  - New icon: 🌐 representing OpenCode's open and universal approach
  - Detection patterns for OpenCode's TUI (input box, mode indicators, logo)
  - Updated all documentation, examples, and tests

## [0.1.0] - 2025-12-03

### Added

- **Terminal UI** - Full-featured TUI built with Bubble Tea
  - Session list with hierarchical group organization
  - Live preview pane showing terminal output
  - Fuzzy search with `/` key
  - Keyboard-driven navigation (vim-style `hjkl`)

- **Session Management**
  - Create, rename, delete sessions
  - Attach/detach with `Ctrl+Q`
  - Import existing tmux sessions
  - Reorder sessions within groups

- **Group Organization**
  - Hierarchical folder structure
  - Create nested groups
  - Move sessions between groups
  - Collapsible groups with persistence

- **Intelligent Status Detection**
  - 3-state model: Running (green), Waiting (yellow), Idle (gray)
  - Tool-specific busy indicator detection
  - Prompt detection for Claude Code, Gemini CLI, OpenCode, Codex
  - Content hashing with 2-second activity cooldown
  - Status persistence across restarts

- **CLI Commands**
  - `agent-deck` - Launch TUI
  - `agent-deck add <path>` - Add session from CLI
  - `agent-deck list` - List sessions (table or JSON)
  - `agent-deck remove <id|title>` - Remove session

- **Tool Support**
  - Claude Code - Full status detection
  - Gemini CLI - Activity and prompt detection
  - OpenCode - TUI element detection
  - Codex - Prompt detection
  - Generic shell support

- **tmux Integration**
  - Automatic session creation with unique names
  - Mouse mode enabled by default
  - 50,000 line scrollback buffer
  - PTY attachment with `Ctrl+Q` detach

### Technical

- Built with Go 1.24+
- Bubble Tea TUI framework
- Lip Gloss styling
- Tokyo Night color theme
- Atomic JSON persistence
- Cross-platform: macOS, Linux

[0.1.0]: https://github.com/asheshgoplani/agent-deck/releases/tag/v0.1.0

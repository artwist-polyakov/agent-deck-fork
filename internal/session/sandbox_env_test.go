package session

import (
	"regexp"
	"strings"
	"testing"
)

// uuidLiteralRE matches a UUID in the format xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
// with only hex chars (lowercase or uppercase). No $( shell expansion.
var uuidLiteralRE = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

// TestCommandBuilders_NoTmuxSetEnv verifies that no command builder produces a shell
// string containing "tmux set-environment". The correct approach is host-side
// SetEnvironment() calls after tmux session start (see instance.go:SyncSessionIDsToTmux).
func TestCommandBuilders_NoTmuxSetEnv(t *testing.T) {
	tests := []struct {
		name      string
		buildFunc func() string
	}{
		{
			name: "buildClaudeCommandWithMessage new session",
			buildFunc: func() string {
				inst := &Instance{
					ID:   "test-inst-id",
					Tool: "claude",
				}
				return inst.buildClaudeCommand("claude")
			},
		},
		{
			name: "buildClaudeCommandWithMessage resume with --session-id",
			buildFunc: func() string {
				inst := &Instance{
					ID:              "test-inst-id",
					Tool:            "claude",
					ClaudeSessionID: "test-session-123",
				}
				// Set resume mode via SetClaudeOptions so buildClaudeCommandWithMessage picks it up
				_ = inst.SetClaudeOptions(&ClaudeOptions{
					SessionMode:     "resume",
					ResumeSessionID: "test-session-123",
				})
				return inst.buildClaudeCommandWithMessage("claude", "")
			},
		},
		{
			name: "buildClaudeCommandWithMessage new session with message",
			buildFunc: func() string {
				inst := &Instance{
					ID:   "test-inst-id",
					Tool: "claude",
				}
				return inst.buildClaudeCommandWithMessage("claude", "hello world")
			},
		},
		{
			name: "buildClaudeResumeCommand",
			buildFunc: func() string {
				inst := &Instance{
					ID:              "test-inst-id",
					Tool:            "claude",
					ClaudeSessionID: "test-resume-session-456",
				}
				return inst.buildClaudeResumeCommand()
			},
		},
		{
			name: "buildGeminiCommand resume",
			buildFunc: func() string {
				yolo := true
				inst := &Instance{
					ID:              "test-inst-id",
					Tool:            "gemini",
					GeminiSessionID: "gemini-session-789",
					GeminiYoloMode:  &yolo,
				}
				return inst.buildGeminiCommand("gemini")
			},
		},
		{
			name: "buildGeminiCommand fresh",
			buildFunc: func() string {
				yolo := false
				inst := &Instance{
					ID:             "test-inst-id",
					Tool:           "gemini",
					GeminiYoloMode: &yolo,
				}
				return inst.buildGeminiCommand("gemini")
			},
		},
		{
			name: "buildOpenCodeCommand resume",
			buildFunc: func() string {
				inst := &Instance{
					ID:                "test-inst-id",
					Tool:              "opencode",
					OpenCodeSessionID: "ses_ABC123",
				}
				return inst.buildOpenCodeCommand("opencode")
			},
		},
		{
			name: "buildCodexCommand resume",
			buildFunc: func() string {
				inst := &Instance{
					ID:             "test-inst-id",
					Tool:           "codex",
					CodexSessionID: "codex-sess-xyz",
				}
				return inst.buildCodexCommand("codex")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.buildFunc()
			if strings.Contains(cmd, "tmux set-environment") {
				t.Errorf("command contains 'tmux set-environment' (must use host-side SetEnvironment instead):\n  cmd: %q", cmd)
			}
		})
	}
}

// TestCommandBuilders_ClaudeResumeCommand_NoSuppressor verifies that buildClaudeResumeCommand
// does not emit the "2>/dev/null" suppressor that was previously added to silence Docker errors.
// After the fix, tmux set-environment is removed entirely, so the suppressor is dead code.
func TestCommandBuilders_ClaudeResumeCommand_NoSuppressor(t *testing.T) {
	inst := &Instance{
		ID:              "test-inst-id",
		Tool:            "claude",
		ClaudeSessionID: "test-resume-session-456",
	}
	cmd := inst.buildClaudeResumeCommand()
	if strings.Contains(cmd, "2>/dev/null") {
		t.Errorf("buildClaudeResumeCommand should not contain '2>/dev/null' (suppressor is dead code after tmux set-environment removal):\n  cmd: %q", cmd)
	}
}

// TestCommandBuilders_NewClaude_LiteralUUID verifies that new Claude sessions embed a
// literal UUID (not a shell expansion like $(uuidgen)) in the --session-id flag.
// Pre-generating in Go avoids the Docker-sandbox failure where uuidgen is unavailable
// and ensures the ID is known to the Instance immediately.
func TestCommandBuilders_NewClaude_LiteralUUID(t *testing.T) {
	inst := &Instance{
		ID:   "test-inst-id",
		Tool: "claude",
	}
	cmd := inst.buildClaudeCommand("claude")

	// Must not use shell expansion for uuidgen
	if strings.Contains(cmd, "$(uuidgen") {
		t.Errorf("new Claude session command uses $(uuidgen) shell expansion; must pre-generate UUID in Go:\n  cmd: %q", cmd)
	}

	// Must contain --session-id with a literal UUID pattern
	if !strings.Contains(cmd, "--session-id") {
		t.Errorf("new Claude session command missing --session-id flag:\n  cmd: %q", cmd)
	}

	// Extract the argument after --session-id and verify it is a literal UUID
	idx := strings.Index(cmd, "--session-id")
	if idx >= 0 {
		rest := strings.TrimSpace(cmd[idx+len("--session-id"):])
		// Strip leading quote if present
		rest = strings.TrimPrefix(rest, `"`)
		rest = strings.TrimPrefix(rest, `'`)
		// Extract up to the first space or quote
		end := strings.IndexAny(rest, ` "'`)
		var candidate string
		if end == -1 {
			candidate = rest
		} else {
			candidate = rest[:end]
		}
		if !uuidLiteralRE.MatchString(candidate) {
			t.Errorf("--session-id argument %q is not a literal UUID (must be pre-generated in Go):\n  cmd: %q", candidate, cmd)
		}
	}
}

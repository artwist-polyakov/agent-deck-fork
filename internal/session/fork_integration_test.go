package session

import (
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestForkFlow_Integration tests the complete fork flow
// This is a longer-running integration test that requires tmux
func TestForkFlow_Integration(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create parent session with claude tool
	parent := NewInstanceWithTool("fork-parent", "/tmp", "claude")

	// Simulate detection: manually set ClaudeSessionID (normally detected from files)
	parentID := "abc-123-def"
	parent.ClaudeSessionID = parentID
	parent.ClaudeDetectedAt = time.Now()
	t.Logf("Parent session ID (simulated detection): %s", parentID)

	// Verify CanFork is true
	if !parent.CanFork() {
		t.Fatal("Parent should be able to fork")
	}

	// Create forked instance
	forked, cmd, err := parent.CreateForkedInstance("fork-child", "")
	if err != nil {
		t.Fatalf("CreateForkedInstance failed: %v", err)
	}

	// Verify fork command structure - uses Go-side UUID (no shell uuidgen dependency)
	// Must NOT use shell uuidgen (replaced with generateUUID() in Go)
	if strings.Contains(cmd, "uuidgen") {
		t.Errorf("Fork command should NOT use shell uuidgen (replaced with Go-side UUID): %s", cmd)
	}
	// CLAUDE_SESSION_ID must NOT be in the shell command string; it is set via host-side SetEnvironment.
	if strings.Contains(cmd, "tmux set-environment CLAUDE_SESSION_ID") {
		t.Errorf("Fork command should NOT embed tmux set-environment (use host-side SetEnvironment): %s", cmd)
	}
	if !strings.Contains(cmd, "AGENTDECK_INSTANCE_ID="+forked.ID) {
		t.Errorf("Fork command should export AGENTDECK_INSTANCE_ID for the forked session: %s", cmd)
	}
	// Should use --session-id with a literal Go-generated UUID (not shell variable)
	if !strings.Contains(cmd, "--session-id") {
		t.Errorf("Fork command should use --session-id flag: %s", cmd)
	}
	if strings.Contains(cmd, `--session-id "$session_id"`) {
		t.Errorf("Fork command should NOT use shell variable for session ID: %s", cmd)
	}
	// Should still use --resume for parent session and --fork-session
	if !strings.Contains(cmd, "--resume "+parentID) {
		t.Errorf("Fork command should have --resume %s: %s", parentID, cmd)
	}
	if !strings.Contains(cmd, "--fork-session") {
		t.Errorf("Fork command should have --fork-session: %s", cmd)
	}
	// Should NOT use capture-resume pattern
	if strings.Contains(cmd, `-p "."`) {
		t.Errorf("Fork command should NOT use -p \".\" capture: %s", cmd)
	}
	if strings.Contains(cmd, "jq") {
		t.Errorf("Fork command should NOT use jq: %s", cmd)
	}

	// Verify forked instance state: ClaudeSessionID should be pre-populated with the Go-generated UUID
	// (the UUID is generated in Go and assigned to the Instance immediately for tracking).
	if forked.ClaudeSessionID == "" {
		t.Errorf("Forked instance should have ClaudeSessionID pre-set by generateUUID()")
	}
	if forked.Tool != "claude" {
		t.Errorf("Forked tool = %s, want claude", forked.Tool)
	}
	if forked.ProjectPath != "/tmp" {
		t.Errorf("Forked path = %s, want /tmp", forked.ProjectPath)
	}

	t.Log("Fork flow test passed - fork command is correctly structured")
}

// TestMultipleSessionsSameProject tests that multiple sessions in same project
// can have different detected session IDs
func TestMultipleSessionsSameProject(t *testing.T) {
	// Create two sessions in the same project directory
	session1 := NewInstanceWithTool("test1", "/tmp/same-project", "claude")
	session2 := NewInstanceWithTool("test2", "/tmp/same-project", "claude")

	// Initially, neither should have session IDs (detection-based)
	if session1.ClaudeSessionID != "" {
		t.Error("session1 should have empty ClaudeSessionID (detection-based)")
	}
	if session2.ClaudeSessionID != "" {
		t.Error("session2 should have empty ClaudeSessionID (detection-based)")
	}

	// Simulate detection with different IDs
	session1.ClaudeSessionID = "abc-123-first"
	session1.ClaudeDetectedAt = time.Now()
	session2.ClaudeSessionID = "def-456-second"
	session2.ClaudeDetectedAt = time.Now()

	// Session IDs should be DIFFERENT
	if session1.ClaudeSessionID == session2.ClaudeSessionID {
		t.Errorf("Sessions in same project should have DIFFERENT IDs: %s == %s",
			session1.ClaudeSessionID, session2.ClaudeSessionID)
	}

	t.Logf("Session 1 ID (detected): %s", session1.ClaudeSessionID)
	t.Logf("Session 2 ID (detected): %s", session2.ClaudeSessionID)
}

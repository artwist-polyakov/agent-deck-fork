package session

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

// skipIfNoOpenCode skips the test if OpenCode CLI is not available
func skipIfNoOpenCode(t *testing.T) {
	t.Helper()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping OpenCode E2E test in CI environment")
	}
	if _, err := exec.LookPath("opencode"); err != nil {
		t.Skip("Skipping: OpenCode CLI not installed")
	}
}

func skipIfNoOpenCodeSessionForProject(t *testing.T, projectPath string) {
	t.Helper()
	probe := &Instance{Tool: "opencode", ProjectPath: projectPath}
	if probe.queryOpenCodeSession() == "" {
		t.Skip("Skipping: no OpenCode sessions available for this project path")
	}
}

func TestOpenCodeDetectionE2E(t *testing.T) {
	if os.Getenv("AGENT_DECK_E2E") == "" {
		t.Skip("Skipping OpenCode E2E test (set AGENT_DECK_E2E=1 to run)")
	}
	skipIfNoOpenCode(t)
	projectPath := "/Users/ashesh/claude-deck"
	skipIfNoOpenCodeSessionForProject(t, projectPath)

	t.Log("=== E2E OpenCode Detection Test ===")

	// Create instance like agent-deck does
	inst := &Instance{
		Tool:        "opencode",
		ProjectPath: projectPath,
	}

	t.Logf("Instance created: Tool=%s, ProjectPath=%s", inst.Tool, inst.ProjectPath)
	t.Logf("Before detection: OpenCodeSessionID=%q", inst.OpenCodeSessionID)

	// Call detection synchronously (not in goroutine)
	t.Log("Calling detectOpenCodeSessionAsync()...")

	// Run detection in goroutine and wait for it
	done := make(chan bool)
	go func() {
		inst.detectOpenCodeSessionAsync()
		done <- true
	}()

	// Wait up to 15 seconds (detection has delays built in)
	select {
	case <-done:
		t.Log("Detection completed")
	case <-time.After(15 * time.Second):
		t.Fatal("Detection timed out after 15 seconds")
	}

	t.Logf("After detection: OpenCodeSessionID=%q", inst.OpenCodeSessionID)
	t.Logf("After detection: OpenCodeDetectedAt=%v", inst.OpenCodeDetectedAt)

	if inst.OpenCodeSessionID == "" {
		t.Error("❌ FAILED: OpenCodeSessionID is empty!")
	} else {
		t.Logf("✅ SUCCESS: Detected session ID: %s", inst.OpenCodeSessionID)
	}
}

// TestQueryOpenCodeSessionDirect tests the query function directly without delays
func TestQueryOpenCodeSessionDirect(t *testing.T) {
	skipIfNoOpenCode(t)
	projectPath := "/Users/ashesh/claude-deck"
	skipIfNoOpenCodeSessionForProject(t, projectPath)

	inst := &Instance{
		Tool:        "opencode",
		ProjectPath: projectPath,
	}

	t.Log("Testing queryOpenCodeSession directly...")

	sessionID := inst.queryOpenCodeSession()

	t.Logf("queryOpenCodeSession returned: %q", sessionID)

	if sessionID == "" {
		t.Error("❌ FAILED: queryOpenCodeSession returned empty string!")
	} else {
		t.Logf("✅ SUCCESS: Got session ID: %s", sessionID)
	}
}

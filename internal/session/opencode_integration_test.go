package session

import (
	"encoding/json"
	"os/exec"
	"testing"
	"time"
)

// TestOpenCodeSessionDetectionIntegration tests the full detection flow
// This test requires OpenCode CLI to be installed
func TestOpenCodeSessionDetectionIntegration(t *testing.T) {
	// Skip if opencode is not installed
	if _, err := exec.LookPath("opencode"); err != nil {
		t.Skip("opencode CLI not installed, skipping integration test")
	}

	projectPath := "/Users/ashesh/claude-deck"

	t.Run("queryOpenCodeSession finds most recent session", func(t *testing.T) {
		// Create a test instance
		inst := &Instance{
			Tool:        "opencode",
			ProjectPath: projectPath,
		}

		// Query for sessions
		sessionID := inst.queryOpenCodeSession()

		if sessionID == "" {
			t.Error("Expected to find a session ID, got empty string")
			t.Log("This might mean there are no OpenCode sessions for this project yet")

			// Let's debug by checking what sessions exist
			cmd := exec.Command("opencode", "session", "list", "--format", "json")
			output, err := cmd.Output()
			if err != nil {
				t.Logf("Failed to list sessions: %v", err)
			} else {
				t.Logf("Available sessions:\n%s", string(output))
			}
			return
		}

		t.Logf("Successfully detected session ID: %s", sessionID)

		// Verify the session ID format (ses_XXXX)
		if len(sessionID) < 4 || sessionID[:4] != "ses_" {
			t.Errorf("Session ID doesn't match expected format ses_XXX: %s", sessionID)
		}
	})

	t.Run("detection returns most recently updated session", func(t *testing.T) {
		// Get all sessions for the project
		cmd := exec.Command("opencode", "session", "list", "--format", "json")
		output, err := cmd.Output()
		if err != nil {
			t.Skipf("Failed to list sessions: %v", err)
		}

		var sessions []struct {
			ID        string `json:"id"`
			Directory string `json:"directory"`
			Updated   int64  `json:"updated"`
		}

		if err := json.Unmarshal(output, &sessions); err != nil {
			t.Fatalf("Failed to parse sessions: %v", err)
		}

		// Find the most recently updated session for our project
		var expectedID string
		var maxUpdated int64
		for _, s := range sessions {
			if s.Directory == projectPath && s.Updated > maxUpdated {
				maxUpdated = s.Updated
				expectedID = s.ID
			}
		}

		if expectedID == "" {
			t.Skip("No sessions found for project path")
		}

		// Now test our detection
		inst := &Instance{
			Tool:        "opencode",
			ProjectPath: projectPath,
		}

		detectedID := inst.queryOpenCodeSession()

		if detectedID != expectedID {
			t.Errorf("Detection returned %s but expected %s (most recently updated)", detectedID, expectedID)
		} else {
			t.Logf("Correctly detected most recent session: %s (updated=%d)", detectedID, maxUpdated)
		}
	})

	t.Run("detection sets OpenCodeSessionID field", func(t *testing.T) {
		inst := &Instance{
			Tool:        "opencode",
			ProjectPath: projectPath,
		}

		// Simulate what detectOpenCodeSessionAsync does (but synchronously)
		sessionID := inst.queryOpenCodeSession()
		if sessionID != "" {
			inst.OpenCodeSessionID = sessionID
			inst.OpenCodeDetectedAt = time.Now()
		}

		if inst.OpenCodeSessionID == "" {
			t.Error("OpenCodeSessionID was not set")
		} else {
			t.Logf("OpenCodeSessionID set to: %s", inst.OpenCodeSessionID)
		}

		if inst.OpenCodeDetectedAt.IsZero() {
			t.Error("OpenCodeDetectedAt was not set")
		} else {
			t.Logf("OpenCodeDetectedAt set to: %s", inst.OpenCodeDetectedAt)
		}
	})
}

// TestOpenCodeCLIAvailable verifies the OpenCode CLI works
func TestOpenCodeCLIAvailable(t *testing.T) {
	// Check if opencode is installed
	path, err := exec.LookPath("opencode")
	if err != nil {
		t.Skipf("opencode CLI not found in PATH: %v", err)
	}
	t.Logf("OpenCode CLI found at: %s", path)

	// Check if we can run session list
	cmd := exec.Command("opencode", "session", "list", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run 'opencode session list': %v", err)
	}

	// Verify it returns valid JSON
	var sessions []interface{}
	if err := json.Unmarshal(output, &sessions); err != nil {
		t.Fatalf("OpenCode session list returned invalid JSON: %v\nOutput: %s", err, string(output))
	}

	t.Logf("OpenCode CLI working, found %d sessions", len(sessions))
}

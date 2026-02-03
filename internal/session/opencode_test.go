package session

import (
	"encoding/json"
	"testing"
)

// TestOpenCodeSessionMatching tests the session matching logic for OpenCode
func TestOpenCodeSessionMatching(t *testing.T) {
	// Mock session data similar to real OpenCode output
	mockSessionsJSON := `[
		{
			"id": "ses_NEW001",
			"title": "New session",
			"updated": 1768982200000,
			"created": 1768982195000,
			"projectId": "60e0658efac270ccb48e12c801746116f86763fa",
			"directory": "/Users/ashesh/claude-deck"
		},
		{
			"id": "ses_OLD001",
			"title": "Old session",
			"updated": 1768982100000,
			"created": 1768982000000,
			"projectId": "60e0658efac270ccb48e12c801746116f86763fa",
			"directory": "/Users/ashesh/claude-deck"
		},
		{
			"id": "ses_OTHER001",
			"title": "Different project",
			"updated": 1768982300000,
			"created": 1768982300000,
			"projectId": "other-project-id",
			"directory": "/Users/ashesh/other-project"
		}
	]`

	tests := []struct {
		name        string
		projectPath string
		startTime   int64
		wantID      string
		wantMatch   bool
	}{
		{
			name:        "Finds most recent session for matching directory",
			projectPath: "/Users/ashesh/claude-deck",
			startTime:   1768982150000, // After OLD, before NEW updated
			wantID:      "ses_NEW001",  // Should pick the most recently updated one
			wantMatch:   true,
		},
		{
			name:        "Picks most recent even when startTime is very recent",
			projectPath: "/Users/ashesh/claude-deck",
			startTime:   1768982500000, // After all sessions
			wantID:      "ses_NEW001",  // Should still pick most recent for directory
			wantMatch:   true,
		},
		{
			name:        "Ignores sessions from different directories",
			projectPath: "/Users/ashesh/other-project",
			startTime:   1768982000000,
			wantID:      "ses_OTHER001", // Only session matching this directory
			wantMatch:   true,
		},
		{
			name:        "No match for unknown directory",
			projectPath: "/Users/ashesh/nonexistent",
			startTime:   1768982000000,
			wantID:      "",
			wantMatch:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse mock sessions
			var sessions []struct {
				ID        string `json:"id"`
				Directory string `json:"directory"`
				Path      string `json:"path"`
				Created   int64  `json:"created"`
				Updated   int64  `json:"updated"`
			}

			if err := json.Unmarshal([]byte(mockSessionsJSON), &sessions); err != nil {
				t.Fatalf("Failed to parse mock sessions: %v", err)
			}

			// Apply the matching logic (same as queryOpenCodeSession but testable)
			gotID := findBestOpenCodeSession(sessions, tt.projectPath)

			if tt.wantMatch {
				if gotID != tt.wantID {
					t.Errorf("Expected session ID %q, got %q", tt.wantID, gotID)
				}
			} else {
				if gotID != "" {
					t.Errorf("Expected no match, got %q", gotID)
				}
			}
		})
	}
}

// findBestOpenCodeSession is a testable version of the session matching logic
// This should match the algorithm used in queryOpenCodeSession()
func findBestOpenCodeSession(sessions []struct {
	ID        string `json:"id"`
	Directory string `json:"directory"`
	Path      string `json:"path"`
	Created   int64  `json:"created"`
	Updated   int64  `json:"updated"`
}, projectPath string) string {
	var bestMatch string
	var bestMatchTime int64

	for _, sess := range sessions {
		// Check directory match (normalize paths)
		sessDir := sess.Directory
		if sessDir == "" {
			sessDir = sess.Path
		}

		normalizedSessDir := normalizePath(sessDir)
		normalizedProjectPath := normalizePath(projectPath)

		if sessDir == "" || normalizedSessDir != normalizedProjectPath {
			continue
		}

		// Pick the most recently updated session for this directory
		// OpenCode automatically resumes the most recent session,
		// so we should track that same session
		updatedAt := sess.Updated
		if updatedAt == 0 {
			updatedAt = sess.Created
		}

		if bestMatch == "" || updatedAt > bestMatchTime {
			bestMatch = sess.ID
			bestMatchTime = updatedAt
		}
	}

	return bestMatch
}

// TestOpenCodeBuildCommand tests the command building for OpenCode sessions
func TestOpenCodeBuildCommand(t *testing.T) {
	tests := []struct {
		name              string
		baseCommand       string
		openCodeSessionID string
		wantContains      []string
		wantNotContains   []string
	}{
		{
			name:              "Fresh start without session ID",
			baseCommand:       "opencode",
			openCodeSessionID: "",
			wantContains:      []string{"opencode"},
			wantNotContains:   []string{"-s", "tmux set-environment"},
		},
		{
			name:              "Resume with existing session ID",
			baseCommand:       "opencode",
			openCodeSessionID: "ses_ABC123",
			wantContains:      []string{"opencode -s ses_ABC123", "tmux set-environment OPENCODE_SESSION_ID ses_ABC123"},
			wantNotContains:   []string{},
		},
		{
			name:              "Custom command passes through unchanged",
			baseCommand:       "opencode --model gpt-4",
			openCodeSessionID: "ses_ABC123",
			wantContains:      []string{"opencode --model gpt-4"},
			wantNotContains:   []string{"-s"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := &Instance{
				Tool:              "opencode",
				OpenCodeSessionID: tt.openCodeSessionID,
			}

			got := inst.buildOpenCodeCommand(tt.baseCommand)

			for _, want := range tt.wantContains {
				if !containsSubstring(got, want) {
					t.Errorf("Expected command to contain %q, got: %q", want, got)
				}
			}

			for _, notWant := range tt.wantNotContains {
				if containsSubstring(got, notWant) {
					t.Errorf("Expected command to NOT contain %q, got: %q", notWant, got)
				}
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

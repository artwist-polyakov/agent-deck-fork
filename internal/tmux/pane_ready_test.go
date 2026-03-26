package tmux

import (
	"testing"
	"time"
)

// TestIsPaneShellReady tests shell prompt detection for various shell types.
func TestIsPaneShellReady(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "BashPrompt",
			input:    "user@host:~$",
			expected: true,
		},
		{
			name:     "BashPromptWithTrailingSpace",
			input:    "user@host:~$ ",
			expected: true,
		},
		{
			name:     "ZshPrompt",
			input:    "user@host ~ %",
			expected: true,
		},
		{
			name:     "ZshPromptWithTrailingSpace",
			input:    "user@host ~ % ",
			expected: true,
		},
		{
			name:     "FishPrompt",
			input:    "user@host ~>",
			expected: true,
		},
		{
			name:     "RootPrompt",
			input:    "root@host:~#",
			expected: true,
		},
		{
			name:     "RootPromptWithTrailingSpace",
			input:    "root@host:~# ",
			expected: true,
		},
		{
			name:     "EmptyOutput",
			input:    "",
			expected: false,
		},
		{
			name:     "OnlyWhitespace",
			input:    "   \n\n  ",
			expected: false,
		},
		{
			name:     "NoPrompt",
			input:    "Loading...",
			expected: false,
		},
		{
			name:     "MultilineWithPromptAtEnd",
			input:    "some output\nmore output\nuser@host:~$ ",
			expected: true,
		},
		{
			name:     "MultilineNoPromptAtEnd",
			input:    "user@host:~$\nsome more output",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPaneShellReady(tt.input)
			if got != tt.expected {
				t.Errorf("isPaneShellReady(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

// TestWaitForPaneReady_Timeout verifies that waitForPaneReady returns an error
// when the timeout expires before a shell prompt is detected.
func TestWaitForPaneReady_Timeout(t *testing.T) {
	skipIfNoTmuxServer(t)

	// Create a real tmux session; on most systems the shell will not be ready
	// within 10ms (especially for the first capture), allowing us to test timeout.
	sess := NewSession("pane-ready-timeout", t.TempDir())
	if err := sess.Start(""); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer func() { _ = sess.Kill() }()

	// 1ms timeout is far too short for any shell to be ready.
	err := waitForPaneReady(sess, 1*time.Millisecond)
	if err == nil {
		// It's technically possible the pane was ready in 1ms on a very fast machine.
		// In that case we skip rather than fail.
		t.Skip("pane was ready in 1ms; skipping timeout assertion")
	}
}

// TestWaitForPaneReady_RealTmux verifies that waitForPaneReady returns nil
// once a shell prompt appears in a freshly-created tmux pane.
func TestWaitForPaneReady_RealTmux(t *testing.T) {
	skipIfNoTmuxServer(t)

	sess := NewSession("pane-ready-real", t.TempDir())
	if err := sess.Start(""); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer func() { _ = sess.Kill() }()

	err := waitForPaneReady(sess, 10*time.Second)
	if err != nil {
		t.Errorf("waitForPaneReady() returned error: %v", err)
	}
}

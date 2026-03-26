package session

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SendSessionMessageReliable sends a message using the same queued/reliable semantics
// as `agent-deck session send` (default mode, no --no-wait).
// It invokes the CLI command to keep behavior identical across callers.
func SendSessionMessageReliable(profile, sessionRef, message string) error {
	sessionRef = strings.TrimSpace(sessionRef)
	message = strings.TrimSpace(message)
	if sessionRef == "" {
		return fmt.Errorf("session reference is required")
	}
	if message == "" {
		return fmt.Errorf("message is required")
	}

	bin := agentDeckBinaryPath()
	args := []string{}
	if strings.TrimSpace(profile) != "" {
		args = append(args, "-p", profile)
	}
	args = append(args, "session", "send", sessionRef, message, "-q")

	cmd := exec.Command(bin, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			return fmt.Errorf("send failed: %w", err)
		}
		return fmt.Errorf("send failed: %s", errMsg)
	}

	return nil
}

func agentDeckBinaryPath() string {
	// In production this should resolve to the installed binary.
	if p := findAgentDeck(); p != "" {
		return p
	}

	// Fall back to current executable only if it looks like the agent-deck binary.
	if exe, err := os.Executable(); err == nil {
		base := strings.ToLower(filepath.Base(exe))
		if strings.HasPrefix(base, "agent-deck") {
			return exe
		}
	}

	return "agent-deck"
}

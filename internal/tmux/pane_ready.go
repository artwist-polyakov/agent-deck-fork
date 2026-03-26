package tmux

import (
	"fmt"
	"strings"
	"time"
)

// isPaneShellReady returns true when the last non-empty line of pane output ends
// with a recognised shell prompt character ($, %, #, >) optionally followed by
// trailing whitespace.
//
// Supported shells:
//   - bash:   ends with $  (e.g. "user@host:~$")
//   - zsh:    ends with %  (e.g. "user@host ~ %")
//   - fish:   ends with >  (e.g. "user@host ~>")
//   - root:   ends with #  (e.g. "root@host:~#")
func isPaneShellReady(output string) bool {
	lines := strings.Split(output, "\n")

	// Walk backwards to find the last non-empty line.
	for i := len(lines) - 1; i >= 0; i-- {
		trimmed := strings.TrimRight(lines[i], " \t")
		if trimmed == "" {
			continue
		}
		last := trimmed[len(trimmed)-1]
		return last == '$' || last == '%' || last == '#' || last == '>'
	}

	// All lines were empty or input was empty.
	return false
}

// waitForPaneReady polls s.CapturePaneFresh() every 100 ms until isPaneShellReady
// returns true or the timeout expires.  Returns nil on success, a descriptive
// error when the timeout fires first.
func waitForPaneReady(s *Session, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		output, err := s.CapturePaneFresh()
		if err == nil && isPaneShellReady(output) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("pane shell not ready after %s", timeout)
}

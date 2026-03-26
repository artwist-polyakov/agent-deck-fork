package logging

import (
	"bytes"
	"log/slog"
	"strings"
)

// BridgeWriter wraps slog as an io.Writer so that legacy log.Printf calls
// flow through the structured logging system. It parses the common
// "[CATEGORY] message" prefix pattern used throughout agent-deck and extracts
// the category into a structured "component" field.
type BridgeWriter struct {
	logger    *slog.Logger
	component string
}

// NewBridgeWriter creates a writer that forwards writes to slog.
// The defaultComponent is used when no [CATEGORY] prefix is found.
func NewBridgeWriter(defaultComponent string) *BridgeWriter {
	return &BridgeWriter{
		logger:    Logger(),
		component: defaultComponent,
	}
}

// Write implements io.Writer. Each write is treated as one log line.
// It strips the standard log timestamp prefix (if present from log.SetFlags)
// and parses [CATEGORY] prefixes into structured fields.
func (bw *BridgeWriter) Write(p []byte) (int, error) {
	n := len(p)
	msg := string(bytes.TrimSpace(p))
	if msg == "" {
		return n, nil
	}

	// Strip standard log timestamp prefix (e.g. "15:04:05.000000 ")
	// The stdlib log package prepends timestamps before writing to the output.
	// Since slog adds its own timestamp, we strip the legacy one.
	msg = stripLogTimestamp(msg)

	// Parse [CATEGORY] prefix
	component := bw.component
	if strings.HasPrefix(msg, "[") {
		if idx := strings.Index(msg, "] "); idx > 0 {
			component = strings.ToLower(msg[1:idx])
			msg = msg[idx+2:]
		}
	}

	// Map known category prefixes to canonical component names
	component = canonicalComponent(component)

	bw.logger.Info(msg, slog.String("component", component))
	return n, nil
}

// stripLogTimestamp removes the time prefix added by log.SetFlags(log.Ltime|log.Lmicroseconds).
// Format: "HH:MM:SS.ffffff " (16 chars).
func stripLogTimestamp(s string) string {
	// log.Ltime|log.Lmicroseconds produces "15:04:05.000000 "
	if len(s) > 16 && s[2] == ':' && s[5] == ':' && s[8] == '.' && s[15] == ' ' {
		return s[16:]
	}
	// log.Ltime produces "15:04:05 "
	if len(s) > 9 && s[2] == ':' && s[5] == ':' && s[8] == ' ' {
		return s[9:]
	}
	return s
}

// canonicalComponent maps known log prefixes to canonical component names.
func canonicalComponent(cat string) string {
	switch cat {
	case "status", "status-debug":
		return CompStatus
	case "mcp", "mcp-debug", "mcp-pool", "mcp-catalog":
		return CompMCP
	case "notif", "notif-bg":
		return CompNotif
	case "perf":
		return CompPerf
	case "pool", "pool-simple", "socket-proxy":
		return CompPool
	case "http", "http-pool", "http-server":
		return CompHTTP
	case "session", "session-data", "restart-debug", "respawn", "opencode", "codex":
		return CompSession
	case "storage":
		return CompStorage
	case "ui":
		return CompUI
	default:
		return cat
	}
}

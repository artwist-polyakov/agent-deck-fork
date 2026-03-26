package mcppool

// ServerStatus represents MCP server state
type ServerStatus int

const (
	StatusStopped ServerStatus = iota
	StatusStarting
	StatusRunning
	StatusFailed
	StatusPermanentlyFailed
)

func (s ServerStatus) String() string {
	switch s {
	case StatusStopped:
		return "stopped"
	case StatusStarting:
		return "starting"
	case StatusRunning:
		return "running"
	case StatusFailed:
		return "failed"
	case StatusPermanentlyFailed:
		return "permanently_failed"
	default:
		return "unknown"
	}
}

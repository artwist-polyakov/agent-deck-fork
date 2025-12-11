package session

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/asheshgoplani/agent-deck/internal/tmux"
	"github.com/google/uuid"
)

// Status represents the current state of a session
type Status string

const (
	StatusRunning Status = "running"
	StatusWaiting Status = "waiting"
	StatusIdle    Status = "idle"
	StatusError   Status = "error"
)

// Instance represents a single agent/shell session
type Instance struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	ProjectPath string        `json:"project_path"`
	GroupPath   string        `json:"group_path"` // e.g., "projects/devops"
	Command     string        `json:"command"`
	Tool        string        `json:"tool"`
	Status      Status        `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`

	// Claude Code integration
	ClaudeSessionID  string    `json:"claude_session_id,omitempty"`
	ClaudeDetectedAt time.Time `json:"claude_detected_at,omitempty"`

	tmuxSession *tmux.Session // Internal tmux session
}

// NewInstance creates a new session instance
func NewInstance(title, projectPath string) *Instance {
	return &Instance{
		ID:          generateID(),
		Title:       title,
		ProjectPath: projectPath,
		GroupPath:   extractGroupPath(projectPath), // Auto-assign group from path
		Tool:        "shell",
		Status:      StatusIdle,
		CreatedAt:   time.Now(),
		tmuxSession: tmux.NewSession(title, projectPath),
	}
}

// NewInstanceWithGroup creates a new session instance with explicit group
func NewInstanceWithGroup(title, projectPath, groupPath string) *Instance {
	inst := NewInstance(title, projectPath)
	inst.GroupPath = groupPath
	return inst
}

// NewInstanceWithTool creates a new session with tool-specific initialization
// When tool is "claude", pre-assigns a ClaudeSessionID for reliable fork support
func NewInstanceWithTool(title, projectPath, tool string) *Instance {
	inst := &Instance{
		ID:          generateID(),
		Title:       title,
		ProjectPath: projectPath,
		GroupPath:   extractGroupPath(projectPath),
		Tool:        tool,
		Status:      StatusIdle,
		CreatedAt:   time.Now(),
		tmuxSession: tmux.NewSession(title, projectPath),
	}

	// Pre-assign Claude session ID if tool is claude
	// This ensures reliable fork functionality by knowing the session ID upfront
	if tool == "claude" {
		inst.ClaudeSessionID = generateUUID()
		inst.ClaudeDetectedAt = time.Now()
	}

	return inst
}

// NewInstanceWithGroupAndTool creates a new session with explicit group and tool
func NewInstanceWithGroupAndTool(title, projectPath, groupPath, tool string) *Instance {
	inst := NewInstanceWithTool(title, projectPath, tool)
	inst.GroupPath = groupPath
	return inst
}

// generateUUID generates a valid UUID v4 for Claude session IDs
func generateUUID() string {
	return uuid.New().String()
}

// extractGroupPath extracts a group path from project path
// e.g., "/home/user/projects/devops" -> "projects"
func extractGroupPath(projectPath string) string {
	parts := strings.Split(projectPath, "/")
	// Find meaningful directory (skip Users, home, etc.)
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		if part != "" && part != "Users" && part != "home" && !strings.HasPrefix(part, ".") {
			// Return parent directory as group if we're at project level
			if i > 0 && i == len(parts)-1 {
				parent := parts[i-1]
				if parent != "" && parent != "Users" && parent != "home" && !strings.HasPrefix(parent, ".") {
					return parent
				}
			}
			return part
		}
	}
	return DefaultGroupName
}

// buildClaudeCommand builds the claude command with --session-id if available
// This ensures reliable session tracking for fork functionality
func (i *Instance) buildClaudeCommand(baseCommand string) string {
	if i.Tool != "claude" || i.ClaudeSessionID == "" {
		return baseCommand
	}

	configDir := GetClaudeConfigDir()

	// If baseCommand is just "claude", build full command with session ID
	if baseCommand == "claude" {
		return fmt.Sprintf("CLAUDE_CONFIG_DIR=%s claude --session-id %s --dangerously-skip-permissions",
			configDir, i.ClaudeSessionID)
	}

	// If it's a custom claude command, inject --session-id after "claude"
	if strings.Contains(baseCommand, "claude") && !strings.Contains(baseCommand, "--session-id") {
		// Find "claude" and insert --session-id right after
		return strings.Replace(baseCommand, "claude ",
			fmt.Sprintf("claude --session-id %s ", i.ClaudeSessionID), 1)
	}

	return baseCommand
}

// Start starts the session in tmux
func (i *Instance) Start() error {
	if i.tmuxSession == nil {
		return fmt.Errorf("tmux session not initialized")
	}

	// Build command with session ID if claude
	command := i.Command
	if i.Tool == "claude" && i.ClaudeSessionID != "" {
		command = i.buildClaudeCommand(i.Command)
	}

	// Start the tmux session
	if err := i.tmuxSession.Start(command); err != nil {
		return fmt.Errorf("failed to start tmux session: %w", err)
	}

	if command != "" {
		i.Status = StatusRunning
	}

	return nil
}

// UpdateStatus updates the session status by checking tmux
func (i *Instance) UpdateStatus() error {
	if i.tmuxSession == nil {
		i.Status = StatusError
		return nil
	}

	// Check if tmux session exists
	if !i.tmuxSession.Exists() {
		i.Status = StatusError
		return nil
	}

	// Get status from tmux session
	status, err := i.tmuxSession.GetStatus()
	if err != nil {
		i.Status = StatusError
		return err
	}

	// Map tmux status to instance status
	switch status {
	case "active":
		i.Status = StatusRunning
	case "waiting":
		i.Status = StatusWaiting
	case "idle":
		i.Status = StatusIdle
	default:
		i.Status = StatusError
	}

	// Update tool detection dynamically (enables fork when Claude starts)
	if detectedTool := i.tmuxSession.DetectTool(); detectedTool != "" {
		i.Tool = detectedTool
	}

	// Update Claude session tracking (non-blocking, best-effort)
	i.UpdateClaudeSession()

	return nil
}

// UpdateClaudeSession updates the Claude session ID if Claude is running
// For sessions with pre-assigned IDs (via --session-id flag), just refresh the timestamp
// For forked sessions and legacy sessions, use detection to find the session ID
func (i *Instance) UpdateClaudeSession() {
	// Only track if tool is Claude
	if i.Tool != "claude" {
		return
	}

	// Check if this is a NEW session with pre-assigned ID (not a fork or legacy session)
	// Pre-assigned sessions have --session-id in their command
	isPREAssigned := i.ClaudeSessionID != "" && strings.Contains(i.Command, "--session-id")

	if isPREAssigned {
		// This is the bulletproof path - we know the ID upfront, no detection needed
		i.ClaudeDetectedAt = time.Now()
		return
	}

	// DETECTION PATH: For forked sessions and legacy sessions
	// Forked sessions use --resume --fork-session (no --session-id)
	// Legacy sessions were created before the --session-id feature

	// Get the session's working directory
	if i.tmuxSession == nil {
		return
	}

	workDir := i.tmuxSession.GetWorkDir()
	if workDir == "" {
		workDir = i.ProjectPath
	}

	// For forked sessions, we need to find the NEW session file created by Claude
	// The fork command creates a new file, so we look for the most recently modified one
	isForkSession := strings.Contains(i.Command, "--fork-session")

	// Try to get session ID from Claude config
	sessionID, err := GetClaudeSessionID(workDir)
	if err != nil {
		// No session found - for forks, this is expected initially (Claude still starting)
		if isForkSession && i.ClaudeSessionID == "" {
			// Don't clear - fork is still initializing, will retry on next tick
			return
		}
		// For non-forks, clear if stale
		if time.Since(i.ClaudeDetectedAt) > 5*time.Minute {
			i.ClaudeSessionID = ""
		}
		return
	}

	// Update session ID
	i.ClaudeSessionID = sessionID
	i.ClaudeDetectedAt = time.Now()
}

// WaitForClaudeSession waits for Claude to create a session file (for forked sessions)
// Returns the detected session ID or empty string after timeout
func (i *Instance) WaitForClaudeSession(maxWait time.Duration) string {
	if i.Tool != "claude" {
		return ""
	}

	workDir := i.ProjectPath
	if i.tmuxSession != nil {
		if wd := i.tmuxSession.GetWorkDir(); wd != "" {
			workDir = wd
		}
	}

	// Poll every 200ms for up to maxWait
	interval := 200 * time.Millisecond
	deadline := time.Now().Add(maxWait)

	for time.Now().Before(deadline) {
		sessionID, err := GetClaudeSessionID(workDir)
		if err == nil && sessionID != "" {
			i.ClaudeSessionID = sessionID
			i.ClaudeDetectedAt = time.Now()
			return sessionID
		}
		time.Sleep(interval)
	}

	return ""
}

// Preview returns the last 3 lines of terminal output
func (i *Instance) Preview() (string, error) {
	if i.tmuxSession == nil {
		return "", fmt.Errorf("tmux session not initialized")
	}

	content, err := i.tmuxSession.CapturePane()
	if err != nil {
		return "", err
	}

	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) > 3 {
		lines = lines[len(lines)-3:]
	}

	return strings.Join(lines, "\n"), nil
}

// PreviewFull returns all terminal output
func (i *Instance) PreviewFull() (string, error) {
	if i.tmuxSession == nil {
		return "", fmt.Errorf("tmux session not initialized")
	}

	return i.tmuxSession.CaptureFullHistory()
}

// HasUpdated checks if there's new output since last check
func (i *Instance) HasUpdated() bool {
	if i.tmuxSession == nil {
		return false
	}

	updated, err := i.tmuxSession.HasUpdated()
	if err != nil {
		return false
	}

	return updated
}

// Kill terminates the tmux session
func (i *Instance) Kill() error {
	if i.tmuxSession == nil {
		return fmt.Errorf("tmux session not initialized")
	}

	if err := i.tmuxSession.Kill(); err != nil {
		return fmt.Errorf("failed to kill tmux session: %w", err)
	}
	i.Status = StatusError
	return nil
}

// Restart recreates the tmux session for a dead/errored session
// This preserves the session ID, title, path, and group but creates a fresh tmux session
func (i *Instance) Restart() error {
	// Create a new tmux session object (keeps same naming convention)
	i.tmuxSession = tmux.NewSession(i.Title, i.ProjectPath)

	// Start the new tmux session
	if err := i.tmuxSession.Start(i.Command); err != nil {
		i.Status = StatusError
		return fmt.Errorf("failed to restart tmux session: %w", err)
	}

	// Update status based on whether we have a command
	if i.Command != "" {
		i.Status = StatusRunning
	} else {
		i.Status = StatusIdle
	}

	return nil
}

// CanRestart returns true if the session can be restarted (is in error state)
func (i *Instance) CanRestart() bool {
	return i.Status == StatusError || i.tmuxSession == nil || !i.tmuxSession.Exists()
}

// CanFork returns true if this session can be forked (has recent Claude session)
func (i *Instance) CanFork() bool {
	if i.ClaudeSessionID == "" {
		return false
	}
	// Session ID must be detected within last 5 minutes
	return time.Since(i.ClaudeDetectedAt) < 5*time.Minute
}

// Fork returns the command to create a forked Claude session
// Returns the claude command string to run in the new tmux session
func (i *Instance) Fork(newTitle, newGroupPath string) (string, error) {
	if !i.CanFork() {
		return "", fmt.Errorf("cannot fork: no active Claude session")
	}

	// Get the actual working directory from tmux (not the stored project_path)
	// Claude uses current directory to locate session files, so we must cd there first
	workDir := i.GetActualWorkDir()

	// Build the fork command with the correct Claude profile
	// This ensures fork uses the same profile where the session ID was detected
	// Uses --dangerously-skip-permissions to match typical cdw workflow
	configDir := GetClaudeConfigDir()
	cmd := fmt.Sprintf("cd %s && CLAUDE_CONFIG_DIR=%s claude --dangerously-skip-permissions --resume %s --fork-session", workDir, configDir, i.ClaudeSessionID)

	return cmd, nil
}

// GetActualWorkDir returns the actual working directory from tmux, or falls back to ProjectPath
func (i *Instance) GetActualWorkDir() string {
	if i.tmuxSession != nil {
		if workDir := i.tmuxSession.GetWorkDir(); workDir != "" {
			return workDir
		}
	}
	return i.ProjectPath
}

// CreateForkedInstance creates a new Instance configured for forking
func (i *Instance) CreateForkedInstance(newTitle, newGroupPath string) (*Instance, string, error) {
	cmd, err := i.Fork(newTitle, newGroupPath)
	if err != nil {
		return nil, "", err
	}

	// Create new instance with the ACTUAL working directory (not stored project_path)
	// This ensures the forked session uses the correct path where Claude session lives
	forked := NewInstance(newTitle, i.GetActualWorkDir())
	if newGroupPath != "" {
		forked.GroupPath = newGroupPath
	} else {
		forked.GroupPath = i.GroupPath
	}
	forked.Command = cmd
	forked.Tool = "claude"

	return forked, cmd, nil
}

// Exists checks if the tmux session still exists
func (i *Instance) Exists() bool {
	if i.tmuxSession == nil {
		return false
	}
	return i.tmuxSession.Exists()
}

// GetTmuxSession returns the tmux session object
func (i *Instance) GetTmuxSession() *tmux.Session {
	return i.tmuxSession
}

// generateID generates a unique session ID
func generateID() string {
	return fmt.Sprintf("%s-%d", randomString(8), time.Now().Unix())
}

// randomString generates a random hex string of specified length
func randomString(length int) string {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

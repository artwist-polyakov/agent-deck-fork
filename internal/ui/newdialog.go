package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// NewDialog represents the new session creation dialog
type NewDialog struct {
	nameInput       textinput.Model
	pathInput       textinput.Model
	commandInput    textinput.Model
	focusIndex      int
	width           int
	height          int
	visible         bool
	presetCommands  []string
	commandCursor   int
	parentGroupPath string
	parentGroupName string
	pathSuggestions []string // stores all available path suggestions
}

// NewNewDialog creates a new NewDialog instance
func NewNewDialog() *NewDialog {
	// Create name input
	nameInput := textinput.New()
	nameInput.Placeholder = "session-name"
	nameInput.Focus()
	nameInput.CharLimit = 100
	nameInput.Width = 40

	// Create path input
	pathInput := textinput.New()
	pathInput.Placeholder = "~/project/path"
	pathInput.CharLimit = 256
	pathInput.Width = 40
	pathInput.ShowSuggestions = true // enable built-in suggestions

	// Get current working directory for default path
	cwd, err := os.Getwd()
	if err == nil {
		pathInput.SetValue(cwd)
	}

	// Create command input
	commandInput := textinput.New()
	commandInput.Placeholder = "custom command"
	commandInput.CharLimit = 100
	commandInput.Width = 40

	return &NewDialog{
		nameInput:       nameInput,
		pathInput:       pathInput,
		commandInput:    commandInput,
		focusIndex:      0,
		visible:         false,
		presetCommands:  []string{"", "claude", "gemini", "aider", "codex"},
		commandCursor:   0,
		parentGroupPath: "default",
		parentGroupName: "default",
	}
}

// ShowInGroup shows the dialog with a pre-selected parent group
func (d *NewDialog) ShowInGroup(groupPath, groupName string) {
	if groupPath == "" {
		groupPath = "default"
		groupName = "default"
	}
	d.parentGroupPath = groupPath
	d.parentGroupName = groupName
	d.visible = true
	d.focusIndex = 0
	d.nameInput.SetValue("")
	d.nameInput.Focus()
	d.commandCursor = 0
}

// GetSelectedGroup returns the parent group path
func (d *NewDialog) GetSelectedGroup() string {
	return d.parentGroupPath
}

// SetSize sets the dialog dimensions
func (d *NewDialog) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// SetPathSuggestions sets the available path suggestions for autocomplete
func (d *NewDialog) SetPathSuggestions(paths []string) {
	d.pathSuggestions = paths
	d.pathInput.SetSuggestions(paths)
}

// Show makes the dialog visible (uses default group)
func (d *NewDialog) Show() {
	d.ShowInGroup("default", "default")
}

// Hide hides the dialog
func (d *NewDialog) Hide() {
	d.visible = false
}

// IsVisible returns whether the dialog is visible
func (d *NewDialog) IsVisible() bool {
	return d.visible
}

// GetValues returns the current dialog values with expanded paths
func (d *NewDialog) GetValues() (name, path, command string) {
	name = strings.TrimSpace(d.nameInput.Value())
	path = strings.TrimSpace(d.pathInput.Value())

	// Expand tilde in path
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[2:])
		}
	}

	// Get command - either from preset or custom input
	if d.commandCursor < len(d.presetCommands) {
		command = d.presetCommands[d.commandCursor]
	}
	if command == "" && d.commandInput.Value() != "" {
		command = strings.TrimSpace(d.commandInput.Value())
	}

	return name, path, command
}

// Validate checks if the dialog values are valid and returns an error message if not
func (d *NewDialog) Validate() string {
	name := strings.TrimSpace(d.nameInput.Value())
	path := strings.TrimSpace(d.pathInput.Value())

	// Check for empty name
	if name == "" {
		return "Session name cannot be empty"
	}

	// Check name length
	if len(name) > 50 {
		return "Session name too long (max 50 characters)"
	}

	// Check for empty path
	if path == "" {
		return "Project path cannot be empty"
	}

	return "" // Valid
}

// updateFocus updates which input has focus
func (d *NewDialog) updateFocus() {
	d.nameInput.Blur()
	d.pathInput.Blur()
	d.commandInput.Blur()

	switch d.focusIndex {
	case 0:
		d.nameInput.Focus()
	case 1:
		d.pathInput.Focus()
	case 2:
		// Command selection (no text input focus needed for presets)
	}
}

// Update handles key messages
func (d *NewDialog) Update(msg tea.Msg) (*NewDialog, tea.Cmd) {
	if !d.visible {
		return d, nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			// On path field: let textinput accept suggestion, then move to next field
			if d.focusIndex == 1 {
				d.pathInput, cmd = d.pathInput.Update(msg)
			}
			// Move to next field
			d.focusIndex = (d.focusIndex + 1) % 3
			d.updateFocus()
			return d, cmd

		case "down":
			// Down always navigates fields (Ctrl+N for suggestions)
			d.focusIndex = (d.focusIndex + 1) % 3
			d.updateFocus()
			return d, nil

		case "shift+tab", "up":
			d.focusIndex--
			if d.focusIndex < 0 {
				d.focusIndex = 2
			}
			d.updateFocus()
			return d, nil

		case "esc":
			d.Hide()
			return d, nil

		case "enter":
			// Let parent handle enter (create session)
			return d, nil

		case "left":
			// Command selection
			if d.focusIndex == 2 {
				d.commandCursor--
				if d.commandCursor < 0 {
					d.commandCursor = len(d.presetCommands) - 1
				}
				return d, nil
			}

		case "right":
			// Command selection
			if d.focusIndex == 2 {
				d.commandCursor = (d.commandCursor + 1) % len(d.presetCommands)
				return d, nil
			}
		}
	}

	// Update focused input
	switch d.focusIndex {
	case 0:
		d.nameInput, cmd = d.nameInput.Update(msg)
	case 1:
		d.pathInput, cmd = d.pathInput.Update(msg)
	}

	return d, cmd
}

// View renders the dialog
func (d *NewDialog) View() string {
	if !d.visible {
		return ""
	}

	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("7"))

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("6")).
		Padding(1, 2).
		Width(60)

	commandBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Margin(0, 1, 0, 0)

	commandBoxSelectedStyle := commandBoxStyle.Copy().
		BorderForeground(lipgloss.Color("6")).
		Bold(true)

	// Active field indicator style
	activeLabelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Bold(true)

	// Build content
	var content strings.Builder

	// Title with parent group info
	content.WriteString(titleStyle.Render("New Session"))
	content.WriteString("\n")
	groupInfoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("5")) // Cyan for group
	content.WriteString(groupInfoStyle.Render("  in group: " + d.parentGroupName))
	content.WriteString("\n\n")

	// Name input
	if d.focusIndex == 0 {
		content.WriteString(activeLabelStyle.Render("▶ Name:"))
	} else {
		content.WriteString(labelStyle.Render("  Name:"))
	}
	content.WriteString("\n")
	content.WriteString("  ")
	content.WriteString(d.nameInput.View())
	content.WriteString("\n\n")

	// Path input
	if d.focusIndex == 1 {
		content.WriteString(activeLabelStyle.Render("▶ Path:"))
	} else {
		content.WriteString(labelStyle.Render("  Path:"))
	}
	content.WriteString("\n")
	content.WriteString("  ")
	content.WriteString(d.pathInput.View())
	content.WriteString("\n")

	// Show path suggestions dropdown when path field is focused
	if d.focusIndex == 1 {
		matched := d.pathInput.MatchedSuggestions()
		if len(matched) > 0 {
			suggestionStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))
			selectedStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("6")).
				Bold(true)
			currentIdx := d.pathInput.CurrentSuggestionIndex()

			// Show up to 5 suggestions
			maxShow := 5
			if len(matched) < maxShow {
				maxShow = len(matched)
			}

			content.WriteString("  ")
			content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("─ suggestions (Tab: accept, Ctrl+N/P: cycle) ─"))
			content.WriteString("\n")

			for i := 0; i < maxShow; i++ {
				style := suggestionStyle
				prefix := "    "
				if i == currentIdx {
					style = selectedStyle
					prefix = "  ▶ "
				}
				content.WriteString(style.Render(prefix + matched[i]))
				content.WriteString("\n")
			}

			if len(matched) > maxShow {
				content.WriteString(suggestionStyle.Render(fmt.Sprintf("    ... and %d more", len(matched)-maxShow)))
				content.WriteString("\n")
			}
		}
	}
	content.WriteString("\n")

	// Command selection
	if d.focusIndex == 2 {
		content.WriteString(activeLabelStyle.Render("▶ Command:"))
	} else {
		content.WriteString(labelStyle.Render("  Command:"))
	}
	content.WriteString("\n  ")

	// Render preset command boxes
	var commandBoxes []string
	for i, cmd := range d.presetCommands {
		displayName := cmd
		if displayName == "" {
			displayName = "shell"
		}

		style := commandBoxStyle
		if i == d.commandCursor && d.focusIndex == 2 {
			style = commandBoxSelectedStyle
		} else if i == d.commandCursor {
			style = commandBoxStyle.Copy().Bold(true)
		}

		commandBoxes = append(commandBoxes, style.Render(displayName))
	}
	content.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, commandBoxes...))
	content.WriteString("\n\n")

	// Custom command input (only if shell is selected)
	if d.commandCursor == 0 {
		content.WriteString(labelStyle.Render("  Custom command:"))
		content.WriteString("\n  ")
		content.WriteString(d.commandInput.View())
		content.WriteString("\n\n")
	}

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		MarginTop(1)
	content.WriteString(helpStyle.Render("Tab: next/accept • ↑↓: navigate • Ctrl+N/P: suggestions • Enter: create • Esc: cancel"))

	// Wrap in dialog box
	dialog := dialogStyle.Render(content.String())

	// Center the dialog
	return lipgloss.Place(
		d.width,
		d.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}

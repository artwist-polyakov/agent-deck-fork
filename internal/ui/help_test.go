package ui

import (
	"strings"
	"testing"
)

func TestHelpOverlayHidesNotesShortcutWhenDisabled(t *testing.T) {
	disabled := false
	setPreviewShowNotesConfigForTest(t, &disabled)

	overlay := NewHelpOverlay()
	overlay.SetSize(100, 40)
	overlay.Show()

	view := overlay.View()
	if strings.Contains(view, "Edit notes") {
		t.Fatalf("help overlay should hide notes shortcut when show_notes=false, got %q", view)
	}
}

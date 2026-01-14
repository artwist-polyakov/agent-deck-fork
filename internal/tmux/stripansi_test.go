package tmux

import (
	"strings"
	"testing"
)

// generateANSIContent creates a test string with many ANSI codes
// to simulate terminal output from Claude or other AI tools
func generateANSIContent(lineCount int) string {
	var b strings.Builder
	for i := 0; i < lineCount; i++ {
		// Simulate typical terminal output with ANSI codes:
		// - Color codes for syntax highlighting
		// - Cursor movement
		// - Bold/italic formatting
		b.WriteString("\x1b[38;5;140m") // Set foreground color
		b.WriteString("Line ")
		b.WriteString("\x1b[1m") // Bold
		b.WriteString("content")
		b.WriteString("\x1b[0m") // Reset
		b.WriteString(" with ")
		b.WriteString("\x1b[32m") // Green
		b.WriteString("colorized")
		b.WriteString("\x1b[0m")
		b.WriteString(" text and ")
		b.WriteString("\x1b[4m") // Underline
		b.WriteString("formatting")
		b.WriteString("\x1b[0m")
		b.WriteString("\n")
	}
	return b.String()
}

// TestStripANSI_Comprehensive verifies the StripANSI function works correctly
func TestStripANSI_Comprehensive(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no ANSI codes",
			input:    "plain text",
			expected: "plain text",
		},
		{
			name:     "simple color code",
			input:    "\x1b[31mred\x1b[0m",
			expected: "red",
		},
		{
			name:     "256 color code",
			input:    "\x1b[38;5;140mfoo\x1b[0m bar",
			expected: "foo bar",
		},
		{
			name:     "multiple codes",
			input:    "\x1b[1m\x1b[31mbold red\x1b[0m normal",
			expected: "bold red normal",
		},
		{
			name:     "cursor movement",
			input:    "\x1b[2Amove up\x1b[2Bmove down",
			expected: "move upmove down",
		},
		{
			name:     "OSC sequence (window title)",
			input:    "\x1b]0;Title\x07content",
			expected: "content",
		},
		{
			name:     "multiline with codes",
			input:    "\x1b[32mline1\x1b[0m\n\x1b[33mline2\x1b[0m",
			expected: "line1\nline2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripANSI(tt.input)
			if result != tt.expected {
				t.Errorf("StripANSI(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// BenchmarkStripANSI_Small benchmarks stripping from a small string
func BenchmarkStripANSI_Small(b *testing.B) {
	content := generateANSIContent(10) // 10 lines
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = StripANSI(content)
	}
}

// BenchmarkStripANSI_Medium benchmarks stripping from medium content
func BenchmarkStripANSI_Medium(b *testing.B) {
	content := generateANSIContent(500) // 500 lines
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = StripANSI(content)
	}
}

// BenchmarkStripANSI_Large benchmarks stripping from large content
// This simulates the 2000-line sessions that caused Issue #39
func BenchmarkStripANSI_Large(b *testing.B) {
	content := generateANSIContent(2000) // 2000 lines - Issue #39 scenario
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = StripANSI(content)
	}
}

// BenchmarkStripANSI_PerLine simulates the OLD pattern of stripping per-line
// This demonstrates why the optimization matters
func BenchmarkStripANSI_PerLine(b *testing.B) {
	content := generateANSIContent(2000)
	lines := strings.Split(content, "\n")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// OLD PATTERN: Strip each line individually (what Issue #39 did)
		for _, line := range lines {
			_ = StripANSI(line)
		}
	}
}

// BenchmarkStripANSI_WholeContent simulates the NEW optimized pattern
func BenchmarkStripANSI_WholeContent(b *testing.B) {
	content := generateANSIContent(2000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// NEW PATTERN: Strip once, then split
		clean := StripANSI(content)
		_ = strings.Split(clean, "\n")
	}
}

// stripANSIOld is the OLD O(n²) implementation that caused Issue #39
// Kept here for benchmark comparison to demonstrate the performance improvement
func stripANSIOld(content string) string {
	result := content

	// Remove CSI sequences (most common): ESC [ ... letter
	for {
		start := strings.Index(result, "\x1b[")
		if start == -1 {
			break
		}
		end := start + 2
		for end < len(result) {
			c := result[end]
			if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
				end++
				break
			}
			end++
		}
		// THIS IS THE BUG: O(n) string copy per ANSI code = O(n²) total
		result = result[:start] + result[end:]
	}

	// Remove OSC sequences: ESC ] ... BEL
	for {
		start := strings.Index(result, "\x1b]")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "\x07")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}

	return result
}

// BenchmarkStripANSI_OldImplementation benchmarks the OLD O(n²) implementation
// This demonstrates how bad performance was before the fix
func BenchmarkStripANSI_OldImplementation(b *testing.B) {
	content := generateANSIContent(2000) // 2000 lines - Issue #39 scenario
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stripANSIOld(content)
	}
}

// BenchmarkStripANSI_OldPerLine simulates the worst-case OLD pattern
// This is what Issue #39 was experiencing: O(n²) × 6000 calls
func BenchmarkStripANSI_OldPerLine(b *testing.B) {
	content := generateANSIContent(2000)
	lines := strings.Split(content, "\n")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// OLD PATTERN: O(n²) strip × 6000 lines = catastrophic
		for _, line := range lines {
			_ = stripANSIOld(line)
		}
	}
}

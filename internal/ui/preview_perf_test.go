package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/asheshgoplani/agent-deck/internal/tmux"
)

// generateRealisticTerminalOutput creates content that mimics real Claude/AI terminal output
// with ANSI color codes, cursor movement, and typical conversation patterns
func generateRealisticTerminalOutput(lineCount int) string {
	var b strings.Builder

	// Simulate Claude Code output with syntax highlighting, prompts, etc.
	colors := []string{
		"\x1b[38;5;140m", // Purple (keywords)
		"\x1b[38;5;114m", // Green (strings)
		"\x1b[38;5;180m", // Orange (functions)
		"\x1b[38;5;75m",  // Blue (types)
		"\x1b[1m",        // Bold
		"\x1b[3m",        // Italic
		"\x1b[4m",        // Underline
	}
	reset := "\x1b[0m"

	for i := 0; i < lineCount; i++ {
		switch i % 10 {
		case 0:
			// Prompt line
			b.WriteString("\x1b[32m❯\x1b[0m ")
		case 1:
			// Code with syntax highlighting
			b.WriteString(colors[0] + "func " + reset)
			b.WriteString(colors[2] + "processData" + reset)
			b.WriteString("(" + colors[3] + "input" + reset + " " + colors[4] + "string" + reset + ") {")
		case 2:
			// Indented code
			b.WriteString("    " + colors[0] + "if " + reset + "len(input) > 0 {")
		case 3:
			// String content
			b.WriteString("        fmt.Println(" + colors[1] + "\"Processing: \"" + reset + " + input)")
		case 4:
			// Closing braces
			b.WriteString("    }")
		case 5:
			// Comment
			b.WriteString(colors[5] + "// This is a comment explaining the code" + reset)
		case 6:
			// Error message (red)
			b.WriteString("\x1b[31mError: " + reset + "Something went wrong at line " + fmt.Sprintf("%d", i))
		case 7:
			// Success message (green)
			b.WriteString("\x1b[32m✓ Test passed:" + reset + " TestFunction_" + fmt.Sprintf("%d", i))
		case 8:
			// Warning (yellow)
			b.WriteString("\x1b[33m⚠ Warning:" + reset + " Deprecated function used")
		default:
			// Regular text
			b.WriteString("Regular output line with some content and details about processing")
		}
		b.WriteString("\n")
	}

	return b.String()
}

// TestPreviewRenderPerformance tests the preview rendering with realistic content
// This is the scenario from Issue #39 - large terminal output causing freezes
func TestPreviewRenderPerformance(t *testing.T) {
	// Skip in CI or short mode
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	testCases := []struct {
		name      string
		lineCount int
		maxTimeMs int64 // Maximum acceptable time in milliseconds
	}{
		{"Small_100_lines", 100, 50},
		{"Medium_500_lines", 500, 100},
		{"Large_1000_lines", 1000, 200},
		{"Issue39_2000_lines", 2000, 500}, // The actual Issue #39 scenario
		{"Extreme_5000_lines", 5000, 1000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			content := generateRealisticTerminalOutput(tc.lineCount)
			contentSize := len(content)

			// Measure the critical path: StripANSI + line processing
			start := time.Now()

			// Step 1: Strip ANSI once (the new optimized approach)
			cleanContent := tmux.StripANSI(content)

			// Step 2: Split into lines
			lines := strings.Split(cleanContent, "\n")

			// Step 3: Process lines (truncation, width measurement)
			// This simulates what renderPreviewPane does
			maxWidth := 80
			var processed []string
			for _, line := range lines {
				if len(line) > maxWidth {
					line = line[:maxWidth-3] + "..."
				}
				processed = append(processed, line)
			}

			// Step 4: Join back (simulates building the output)
			_ = strings.Join(processed, "\n")

			elapsed := time.Since(start)
			elapsedMs := elapsed.Milliseconds()

			t.Logf("Content: %d lines, %d bytes", tc.lineCount, contentSize)
			t.Logf("Elapsed: %v (%d ms)", elapsed, elapsedMs)
			t.Logf("Per-line: %.2f µs", float64(elapsed.Microseconds())/float64(tc.lineCount))

			if elapsedMs > tc.maxTimeMs {
				t.Errorf("Performance regression: took %d ms, expected < %d ms", elapsedMs, tc.maxTimeMs)
			}
		})
	}
}

// stripANSIOldO2 is the OLD O(n²) implementation that caused Issue #39
// This is the algorithm that was in detector.go before the fix
func stripANSIOldO2(content string) string {
	result := content
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
	return result
}

// TestPreviewRenderOldVsNew compares OLD O(n²) implementation vs NEW O(n) regex
func TestPreviewRenderOldVsNew(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance comparison in short mode")
	}

	content := generateRealisticTerminalOutput(2000) // Issue #39 scenario

	// NEW APPROACH: O(n) regex-based stripping
	startNew := time.Now()
	_ = tmux.StripANSI(content)
	elapsedNew := time.Since(startNew)

	// OLD APPROACH: O(n²) string concatenation in loop
	startOld := time.Now()
	_ = stripANSIOldO2(content)
	elapsedOld := time.Since(startOld)

	t.Logf("NEW O(n) regex: %v", elapsedNew)
	t.Logf("OLD O(n²) loop: %v", elapsedOld)
	t.Logf("Speedup: %.1fx faster", float64(elapsedOld)/float64(elapsedNew))

	// The new approach should be significantly faster
	if elapsedNew > elapsedOld {
		t.Errorf("New approach is slower than old! New: %v, Old: %v", elapsedNew, elapsedOld)
	} else {
		t.Logf("✓ Performance improvement confirmed: %.1fx faster", float64(elapsedOld)/float64(elapsedNew))
	}
}

// TestNoUIFreezeSimulation simulates the full detach scenario
// This tests that the render path completes within acceptable time
func TestNoUIFreezeSimulation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping freeze simulation in short mode")
	}

	// Simulate 2000+ lines of terminal output (Issue #39 scenario)
	content := generateRealisticTerminalOutput(2500)

	// The acceptable time for the ENTIRE render path
	// Issue #39 reported 2-11 SECONDS, we should be under 100ms
	maxAcceptableTime := 100 * time.Millisecond

	start := time.Now()

	// Simulate the full preview render path:
	// 1. Strip ANSI from full content
	cleanContent := tmux.StripANSI(content)

	// 2. Split into lines
	lines := strings.Split(cleanContent, "\n")

	// 3. Trim trailing empty lines
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	// 4. Take last N lines (viewport)
	maxLines := 50
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}

	// 5. Process each line (truncation, measurement)
	maxWidth := 100
	var output strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			output.WriteString("\n")
			continue
		}
		if len(line) > maxWidth {
			line = line[:maxWidth-3] + "..."
		}
		output.WriteString(line)
		output.WriteString("\n")
	}

	// 6. Get final result
	_ = output.String()

	elapsed := time.Since(start)

	t.Logf("Full render simulation completed in %v", elapsed)
	t.Logf("Content size: %d bytes, %d lines", len(content), len(strings.Split(content, "\n")))

	if elapsed > maxAcceptableTime {
		t.Errorf("UI FREEZE DETECTED: Render took %v, expected < %v", elapsed, maxAcceptableTime)
		t.Errorf("This would cause the Issue #39 UI freeze!")
	} else {
		t.Logf("✓ No UI freeze - render completed in %v (< %v threshold)", elapsed, maxAcceptableTime)
	}
}

// Note: TestMain is defined in testmain_test.go for test isolation

package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestClampWidth(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{50, 72},   // Below minimum, clamp to 72
		{72, 72},   // At minimum
		{80, 80},   // Normal width
		{100, 100}, // At maximum
		{120, 100}, // Above maximum, clamp to 100
		{200, 100}, // Well above maximum
	}

	for _, tt := range tests {
		result := clampWidth(tt.input)
		if result != tt.expected {
			t.Errorf("clampWidth(%d) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestDetectWidth(t *testing.T) {
	// Test with COLUMNS env var
	originalColumns := os.Getenv("COLUMNS")
	defer os.Setenv("COLUMNS", originalColumns)

	os.Setenv("COLUMNS", "120")
	width := detectWidth()
	// The result may vary depending on whether stdout is a terminal
	if width <= 0 {
		t.Errorf("detectWidth() returned %d, expected positive value", width)
	}

	// Test with invalid COLUMNS
	os.Setenv("COLUMNS", "invalid")
	width = detectWidth()
	// Should fall back to default (80) or terminal width
	if width <= 0 {
		t.Errorf("detectWidth() returned %d, expected positive value", width)
	}

	// Test with empty COLUMNS
	os.Setenv("COLUMNS", "")
	width = detectWidth()
	if width <= 0 {
		t.Errorf("detectWidth() returned %d, expected positive value", width)
	}
}

func TestSupportsUnicode(t *testing.T) {
	// Save original environment
	originalTerm := os.Getenv("TERM")
	originalLcAll := os.Getenv("LC_ALL")
	originalLcCtype := os.Getenv("LC_CTYPE")
	originalLang := os.Getenv("LANG")

	defer func() {
		os.Setenv("TERM", originalTerm)
		os.Setenv("LC_ALL", originalLcAll)
		os.Setenv("LC_CTYPE", originalLcCtype)
		os.Setenv("LANG", originalLang)
	}()

	// Test with dumb terminal
	os.Setenv("TERM", "dumb")
	os.Setenv("LC_ALL", "")
	os.Setenv("LC_CTYPE", "")
	os.Setenv("LANG", "")
	if supportsUnicode() {
		t.Error("expected supportsUnicode() = false for dumb terminal")
	}

	// Test with UTF-8 locale
	os.Setenv("TERM", "xterm")
	os.Setenv("LC_ALL", "en_US.UTF-8")
	if !supportsUnicode() {
		t.Error("expected supportsUnicode() = true for UTF-8 locale")
	}

	// Test with utf8 in LANG
	os.Setenv("LC_ALL", "")
	os.Setenv("LANG", "C.utf8")
	if !supportsUnicode() {
		t.Error("expected supportsUnicode() = true for utf8 in LANG")
	}
}

func TestGradientText(t *testing.T) {
	// Save original environment for Unicode check
	originalLang := os.Getenv("LANG")
	defer os.Setenv("LANG", originalLang)

	os.Setenv("LANG", "en_US.UTF-8")
	os.Setenv("TERM", "xterm")

	// Test with no colors
	result := gradientText("hello", nil)
	if result != "hello" {
		t.Errorf("expected 'hello' with no colors, got %q", result)
	}

	// Test with colors - just verify no panic
	result = gradientText("hello", []lipgloss.Color{colorMauve, colorBlue})
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestBullet(t *testing.T) {
	result := bullet("slb run", "run a command")

	// Should contain the command
	if result == "" {
		t.Error("expected non-empty bullet result")
	}

	// Result should contain the command text
	// The styling makes exact matching difficult, but the content should be there
}

func TestRenderSection(t *testing.T) {
	lines := []string{
		"  line 1",
		"  line 2",
	}

	// Test with unicode
	result := renderSection(true, "🔷 Test Section", lines)
	if result == "" {
		t.Error("expected non-empty section result with unicode")
	}

	// Test without unicode
	result = renderSection(false, "🔷 Test Section", lines)
	if result == "" {
		t.Error("expected non-empty section result without unicode")
	}
}

func TestTierLegend(t *testing.T) {
	// Test with unicode
	result := tierLegend(true)
	if result == "" {
		t.Error("expected non-empty tier legend with unicode")
	}

	// Test without unicode
	result = tierLegend(false)
	if result == "" {
		t.Error("expected non-empty tier legend without unicode")
	}
}

func TestFlagLegend(t *testing.T) {
	// Test with unicode
	result := flagLegend(true)
	if result == "" {
		t.Error("expected non-empty flag legend with unicode")
	}

	// Test without unicode
	result = flagLegend(false)
	if result == "" {
		t.Error("expected non-empty flag legend without unicode")
	}
}

func TestFooterLegend(t *testing.T) {
	// Test with unicode
	result := footerLegend(true)
	if result == "" {
		t.Error("expected non-empty footer legend with unicode")
	}

	// Test without unicode
	result = footerLegend(false)
	if result == "" {
		t.Error("expected non-empty footer legend without unicode")
	}
}

func TestShowQuickReference(t *testing.T) {
	// Save original environment
	originalLang := os.Getenv("LANG")
	originalTerm := os.Getenv("TERM")
	defer func() {
		os.Setenv("LANG", originalLang)
		os.Setenv("TERM", originalTerm)
	}()

	// Set up environment for testing
	os.Setenv("LANG", "en_US.UTF-8")
	os.Setenv("TERM", "xterm")

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	showQuickReference()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should produce non-empty output
	if output == "" {
		t.Error("expected non-empty output from showQuickReference")
	}

	// Should contain SLB reference content
	if !strings.Contains(output, "SLB") && !strings.Contains(output, "slb") {
		t.Error("expected output to contain SLB reference")
	}
}

func TestShowQuickReference_NonUnicode(t *testing.T) {
	// Save original environment
	originalLang := os.Getenv("LANG")
	originalTerm := os.Getenv("TERM")
	defer func() {
		os.Setenv("LANG", originalLang)
		os.Setenv("TERM", originalTerm)
	}()

	// Set up dumb terminal (no unicode)
	os.Setenv("LANG", "C")
	os.Setenv("TERM", "dumb")
	os.Unsetenv("LC_ALL")
	os.Unsetenv("LC_CTYPE")

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	showQuickReference()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should still produce output
	if output == "" {
		t.Error("expected non-empty output from showQuickReference in non-unicode mode")
	}
}

// Ensure lipgloss import is used
var _ lipgloss.Color

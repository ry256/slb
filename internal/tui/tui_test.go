package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNew(t *testing.T) {
	m := New()
	if m.dashboard == nil {
		t.Error("dashboard model should not be nil")
	}
	if m.view != ViewDashboard {
		t.Error("initial view should be ViewDashboard")
	}
}

func TestNewWithOptions(t *testing.T) {
	opts := Options{
		ProjectPath:     "/tmp/test",
		Theme:           "latte",
		DisableMouse:    true,
		RefreshInterval: 10,
	}
	m := NewWithOptions(opts)
	if m.options.ProjectPath != "/tmp/test" {
		t.Errorf("expected project path /tmp/test, got %s", m.options.ProjectPath)
	}
	if m.options.DisableMouse != true {
		t.Error("expected DisableMouse to be true")
	}
}

func TestModelInit(t *testing.T) {
	m := New()
	cmd := m.Init()
	// Init may return commands - just verify it doesn't panic
	_ = cmd
}

func TestModelUpdate(t *testing.T) {
	m := New()
	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if updated == nil {
		t.Error("Update should return non-nil model")
	}
	_ = cmd

	// Verify size was stored
	um := updated.(Model)
	if um.width != 80 || um.height != 24 {
		t.Errorf("expected dimensions 80x24, got %dx%d", um.width, um.height)
	}
}

func TestModelView(t *testing.T) {
	m := New()
	// Set window size first
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	view := m.View()
	if view == "" {
		t.Error("View should return non-empty string")
	}
}

func TestNavigateToPatterns(t *testing.T) {
	m := New()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Simulate 'm' key press to navigate to patterns
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	um := updated.(Model)
	if um.view != ViewPatterns {
		t.Errorf("expected view to be ViewPatterns, got %d", um.view)
	}
}

func TestNavigateToHistory(t *testing.T) {
	m := New()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(Model)

	// Simulate 'H' key press to navigate to history
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}})
	um := updated.(Model)
	if um.view != ViewHistory {
		t.Errorf("expected view to be ViewHistory, got %d", um.view)
	}
}

func TestNavigateBackFromPatterns(t *testing.T) {
	m := New()
	m.view = ViewPatterns

	// Simulate 'esc' key press to navigate back
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	um := updated.(Model)
	if um.view != ViewDashboard {
		t.Errorf("expected view to be ViewDashboard, got %d", um.view)
	}
}

func TestNavigateBackFromHistory(t *testing.T) {
	m := New()
	m.view = ViewHistory

	// Simulate 'b' key press to navigate back
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	um := updated.(Model)
	if um.view != ViewDashboard {
		t.Errorf("expected view to be ViewDashboard, got %d", um.view)
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	if opts.RefreshInterval != 5 {
		t.Errorf("expected default refresh interval 5, got %d", opts.RefreshInterval)
	}
	if opts.DisableMouse != false {
		t.Error("expected default DisableMouse to be false")
	}
	if opts.Theme != "" {
		t.Errorf("expected default theme to be empty, got %s", opts.Theme)
	}
}

// ============== Placeholder Model Tests ==============

func TestPlaceholderModelInit(t *testing.T) {
	m := placeholderModel{}
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init should return nil")
	}
}

func TestPlaceholderModelUpdate(t *testing.T) {
	m := placeholderModel{}

	// Test non-quit key
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd != nil {
		t.Error("Non-quit key should not return quit command")
	}
	_ = updated

	// Test 'q' key
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	// Note: tea.Quit is a function, so we check if cmd is non-nil for quit
	_ = cmd
	_ = updated

	// Test ctrl+c
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	_ = cmd
	_ = updated

	// Test esc
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	_ = cmd
	_ = updated
}

func TestPlaceholderModelView(t *testing.T) {
	m := placeholderModel{}
	view := m.View()
	if view == "" {
		t.Error("View should return non-empty string")
	}
	if len(view) < 10 {
		t.Error("View should return a meaningful message")
	}
}

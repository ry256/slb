// Package tui implements the Bubble Tea terminal UI for SLB.
// Uses the Charmbracelet ecosystem: Bubble Tea, Bubbles, Lip Gloss, Glamour.
package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Dicklesworthstone/slb/internal/tui/dashboard"
)

// Model represents the main TUI model.
type Model struct {
	inner tea.Model
}

// New creates a new TUI model.
func New() Model {
	return Model{inner: dashboard.New("")}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	if m.inner == nil {
		return nil
	}
	return m.inner.Init()
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.inner == nil {
		return m, nil
	}
	next, cmd := m.inner.Update(msg)
	m.inner = next
	return m, cmd
}

// View implements tea.Model.
func (m Model) View() string {
	if m.inner == nil {
		return "Loading..."
	}
	return m.inner.View()
}

// Run starts the TUI.
func Run() error {
	p := tea.NewProgram(New(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

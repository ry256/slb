// Package request provides TUI views for request management.
package request

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Dicklesworthstone/slb/internal/db"
	"github.com/Dicklesworthstone/slb/internal/tui/components"
	"github.com/Dicklesworthstone/slb/internal/tui/theme"
)

// RejectKeyMap defines keybindings for the rejection form.
type RejectKeyMap struct {
	Submit key.Binding
	Cancel key.Binding
}

// DefaultRejectKeyMap returns the default keybindings.
func DefaultRejectKeyMap() RejectKeyMap {
	return RejectKeyMap{
		Submit: key.NewBinding(
			key.WithKeys("ctrl+s", "ctrl+enter"),
			key.WithHelp("ctrl+s", "submit"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}
}

// RejectModel is the Bubble Tea model for the rejection form.
type RejectModel struct {
	Request   *db.Request
	Width     int
	Height    int
	KeyMap    RejectKeyMap
	Submitted bool
	Cancelled bool

	// Form fields
	Reason      string
	reasonInput textarea.Model

	// Validation
	showError bool
	errorMsg  string
}

// NewRejectModel creates a new rejection form model.
func NewRejectModel(request *db.Request) *RejectModel {
	// Create textarea for reason
	ti := textarea.New()
	ti.Placeholder = "Explain why you are rejecting this request..."
	ti.ShowLineNumbers = false
	ti.SetHeight(4)
	ti.Focus()

	return &RejectModel{
		Request:     request,
		KeyMap:      DefaultRejectKeyMap(),
		reasonInput: ti,
	}
}

// Init initializes the model.
func (m *RejectModel) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles messages.
func (m *RejectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.reasonInput.SetWidth(m.Width - 8)
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.Submit):
			reason := strings.TrimSpace(m.reasonInput.Value())
			if reason == "" {
				m.showError = true
				m.errorMsg = "A reason is required when rejecting a request"
				return m, nil
			}
			m.Reason = reason
			m.Submitted = true
			return m, nil

		case key.Matches(msg, m.KeyMap.Cancel):
			m.Cancelled = true
			return m, nil
		}

		// Clear error on typing
		if m.showError {
			m.showError = false
		}
	}

	// Update textarea
	var taCmd tea.Cmd
	m.reasonInput, taCmd = m.reasonInput.Update(msg)
	cmds = append(cmds, taCmd)

	return m, tea.Batch(cmds...)
}

// View renders the model.
func (m *RejectModel) View() string {
	th := theme.Current
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(th.Red).
		Bold(true).
		Padding(1, 0)
	b.WriteString(titleStyle.Render("Reject Request"))
	b.WriteString("\n\n")

	// Request summary
	summaryStyle := lipgloss.NewStyle().
		Foreground(th.Subtext).
		Padding(0, 2)

	cmdPreview := m.Request.Command.Raw
	if len(cmdPreview) > 60 {
		cmdPreview = cmdPreview[:60] + "..."
	}

	tierBadge := components.RenderRiskIndicator(string(m.Request.RiskTier))

	summary := lipgloss.JoinVertical(lipgloss.Left,
		"Request: "+m.Request.ID,
		"Command: "+cmdPreview,
		"Tier: "+tierBadge,
	)
	b.WriteString(summaryStyle.Render(summary))
	b.WriteString("\n\n")

	// Warning message
	warnStyle := lipgloss.NewStyle().
		Foreground(th.Yellow).
		Bold(true).
		Padding(0, 2)

	warnMsg := "You are about to REJECT this request.\nThe command will NOT be executed."
	b.WriteString(warnStyle.Render(warnMsg))
	b.WriteString("\n\n")

	// Reason input (required)
	labelStyle := lipgloss.NewStyle().
		Foreground(th.Blue).
		Bold(true).
		Padding(0, 2)

	requiredStyle := lipgloss.NewStyle().
		Foreground(th.Red).
		Bold(true)

	b.WriteString(labelStyle.Render("Reason ") + requiredStyle.Render("(required)") + labelStyle.Render(":"))
	b.WriteString("\n")

	// Input border color changes on error
	borderColor := th.Overlay0
	if m.showError {
		borderColor = th.Red
	}

	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Margin(0, 2)
	b.WriteString(inputStyle.Render(m.reasonInput.View()))

	// Show error message
	if m.showError {
		errorStyle := lipgloss.NewStyle().
			Foreground(th.Red).
			Padding(0, 2)
		b.WriteString("\n" + errorStyle.Render(m.errorMsg))
	}
	b.WriteString("\n\n")

	// Footer with keybindings
	footerStyle := lipgloss.NewStyle().
		Foreground(th.Subtext).
		Padding(0, 2)

	keyStyle := lipgloss.NewStyle().Foreground(th.Mauve).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(th.Subtext)

	footer := keyStyle.Render("[ctrl+s]") + descStyle.Render(" submit") + "  " +
		keyStyle.Render("[esc]") + descStyle.Render(" cancel")
	b.WriteString(footerStyle.Render(footer))

	// Wrap in a panel
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Red).
		Padding(1, 2).
		Width(m.Width - 4)

	return panelStyle.Render(b.String())
}

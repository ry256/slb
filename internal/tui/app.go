package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// RunPlaceholder launches a minimal placeholder TUI.
// This avoids clashing with the main Run in tui/tui.go.
func RunPlaceholder(ctx context.Context) error {
	model := placeholderModel{}
	p := tea.NewProgram(model)
	_, err := p.Run()
	return err
}

type placeholderModel struct{}

func (placeholderModel) Init() tea.Cmd { return nil }

func (placeholderModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		switch m.String() {
		case "q", "ctrl+c", "esc":
			return placeholderModel{}, tea.Quit
		}
	}
	return placeholderModel{}, nil
}

func (placeholderModel) View() string {
	return fmt.Sprintf("slb TUI is coming soon.\nPress q to exit.\n")
}

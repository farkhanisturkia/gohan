package utils

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true)
	activeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("36")).Bold(true) // Cyan
	inactiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)  // Green
)

type selectModel struct {
	label    string
	items    []string
	cursor   int
	choice   string
	canceled bool
}

func (m selectModel) Init() tea.Cmd {
	return nil
}

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.canceled = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}

		case "enter":
			m.choice = m.items[m.cursor]
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m selectModel) View() string {
	if m.canceled {
		return ""
	}

	if m.choice != "" {
		return fmt.Sprintf("%s %s\n", selectedStyle.Render("✔"), titleStyle.Render(m.choice))
	}

	s := fmt.Sprintf("%s\n", titleStyle.Render(m.label))

	for i, item := range m.items {
		if m.cursor == i {
			s += fmt.Sprintf("  %s %s\n", activeStyle.Render(">"), activeStyle.Render(item))
		} else {
			s += fmt.Sprintf("    %s\n", inactiveStyle.Render(item))
		}
	}

	return s
}

func RunSelect(label string, items []string) (string, error) {
	p := tea.NewProgram(selectModel{
		label: label,
		items: items,
	})

	m, err := p.Run()
	if err != nil {
		return "", err
	}

	finalModel := m.(selectModel)
	if finalModel.canceled {
		return "", fmt.Errorf("initialization cancelled")
	}

	return finalModel.choice, nil
}
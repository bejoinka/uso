package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type item struct {
	label    string
	selected bool
}

type model struct {
	items  []item
	cursor int
	done   bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case " ":
			m.items[m.cursor].selected = !m.items[m.cursor].selected
		case "enter":
			m.done = true
			return m, tea.Quit
		case "q", "ctrl+c":
			m.items = nil // signal cancellation
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder
	for i, it := range m.items {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		check := "[ ]"
		if it.selected {
			check = "[x]"
		}
		fmt.Fprintf(&b, "%s%s %s\n", cursor, check, it.label)
	}
	b.WriteString("\n  arrows move, space toggles, enter confirms\n")
	return b.String()
}

// MultiSelect runs an interactive multi-select and returns the selected indices.
// All items are selected by default.
func MultiSelect(labels []string) ([]int, error) {
	items := make([]item, len(labels))
	for i, l := range labels {
		items[i] = item{label: l, selected: true}
	}

	m := model{items: items}
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	final := finalModel.(model)
	if final.items == nil {
		return nil, fmt.Errorf("cancelled")
	}

	var selected []int
	for i, it := range final.items {
		if it.selected {
			selected = append(selected, i)
		}
	}
	return selected, nil
}

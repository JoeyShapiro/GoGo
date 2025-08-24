package main

// A simple example that shows how to retrieve a value from a Bubble Tea
// program after the Bubble Tea has exited.

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var choices = []string{"Computer", "Player"}

type ModelCreate struct {
	cursor int
	Id     string
}

func (m ModelCreate) Init() tea.Cmd {
	return nil
}

func (m ModelCreate) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc":
			m.Id = ""
			return m, nil

		case "enter":
			// clear the state so the view can go back to menu
			id := m.Id
			return m, func() tea.Msg {
				return CreateMsg{
					GameId:    id,
					AgainstAI: m.cursor == 0,
				}
			}

		case "down", "j":
			m.cursor++
			if m.cursor >= len(choices) {
				m.cursor = 0
			}

		case "up", "k":
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(choices) - 1
			}
		}
	}

	return m, nil
}

func (m ModelCreate) View() string {
	s := strings.Builder{}
	s.WriteString("Play against the computer or another player?\n\n")

	for i := range choices {
		if m.cursor == i {
			s.WriteString("(•) ")
		} else {
			s.WriteString("( ) ")
		}
		s.WriteString(choices[i])
		s.WriteString("\n")
	}
	s.WriteString("\n(press q to quit)\n")

	return s.String()
}

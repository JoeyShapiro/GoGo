package main

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func newItemDelegate(keys *delegateKeyMap) list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.Styles = list.NewDefaultItemStyles()

	d.UpdateFunc = func(msg tea.Msg, m *list.Model) tea.Cmd {
		var title string

		if i, ok := m.SelectedItem().(item); ok {
			title = i.Title()
		} else {
			return nil
		}

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.join):
				return m.NewStatusMessage(statusMessageStyle("You chose " + title))
			}
		}

		return nil
	}

	help := []key.Binding{keys.join}

	d.ShortHelpFunc = func() []key.Binding {
		return help
	}

	d.FullHelpFunc = func() [][]key.Binding {
		return [][]key.Binding{help}
	}

	return d
}

func DefaultItemStyles() (s list.DefaultItemStyles) {
	s.NormalTitle = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#ffffff"}).
		Padding(0, 0, 0, 2) //nolint:mnd

	s.NormalDesc = s.NormalTitle.
		Foreground(lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#ffffff"})

	s.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#ffffff"}).
		Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#ffffff"}).
		Padding(0, 0, 0, 1)

	s.SelectedDesc = s.SelectedTitle.
		Foreground(lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#ffffff"})

	s.DimmedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#ffffff"}).
		Padding(0, 0, 0, 2) //nolint:mnd

	s.DimmedDesc = s.DimmedTitle.
		Foreground(lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#ffffff"})

	s.FilterMatch = lipgloss.NewStyle().Underline(true)

	return s
}

type delegateKeyMap struct {
	join key.Binding
}

// Additional short help entries. This satisfies the help.KeyMap interface and
// is entirely optional.
func (d delegateKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		d.join,
	}
}

// Additional full help entries. This satisfies the help.KeyMap interface and
// is entirely optional.
func (d delegateKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			d.join,
		},
	}
}

func newDelegateKeyMap() *delegateKeyMap {
	return &delegateKeyMap{
		join: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "join game"),
		),
	}
}

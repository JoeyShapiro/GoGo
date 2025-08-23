package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ModelMenu struct {
	term         string
	width        int
	height       int
	txtStyle     lipgloss.Style
	renderer     *lipgloss.Renderer
	Id           int
	list         list.Model
	keys         *listKeyMap
	delegateKeys *delegateKeyMap
}

var (
	appStyle = lipgloss.NewStyle().Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1)
)

type item struct {
	title       string
	description string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.description }
func (i item) FilterValue() string { return i.title }

type listKeyMap struct {
	togglePagination key.Binding
	createGame       key.Binding
	refreshGames     key.Binding
}

func newListKeyMap() *listKeyMap {
	return &listKeyMap{
		togglePagination: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "toggle pagination"),
		),
		createGame: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("C", "create game"),
		),
		refreshGames: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "refresh games"),
		),
	}
}

func newModel(renderer *lipgloss.Renderer) ModelMenu {
	var (
		delegateKeys = newDelegateKeyMap()
		listKeys     = newListKeyMap()
	)

	// Make initial list of items
	items := []list.Item{}
	for _, game := range games {
		items = append(items, item{
			title:       game.Id,
			description: fmt.Sprintf("Players: %d; Turns: %d; Size: %d", game.Players, len(game.Moves), game.BoardSize),
		})
	}

	// Setup list
	delegate := newItemDelegate(delegateKeys, renderer)
	gamesList := list.New(items, delegate, 0, 0)
	gamesList.Title = "Current Games"
	gamesList.Styles.Title = titleStyle
	gamesList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listKeys.togglePagination,
			listKeys.createGame,
			listKeys.refreshGames,
		}
	}

	return ModelMenu{
		renderer:     renderer,
		list:         gamesList,
		keys:         listKeys,
		delegateKeys: delegateKeys,
	}
}

func (m ModelMenu) Init() tea.Cmd {
	return nil
}

func (m ModelMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

	case tea.KeyMsg:
		// Don't match any of the keys below if we're actively filtering.
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, m.keys.togglePagination):
			m.list.SetShowPagination(!m.list.ShowPagination())
			return m, nil

		case key.Matches(msg, m.keys.createGame):
			statusMessageStyle := m.renderer.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#04B575"}).
				Render

			m.delegateKeys.join.SetEnabled(true)
			newItem := item{
				title:       "New Item",
				description: "This is a new item.",
			}
			insCmd := m.list.InsertItem(0, newItem)
			statusCmd := m.list.NewStatusMessage(statusMessageStyle("Added " + newItem.Title()))
			return m, tea.Batch(insCmd, statusCmd)

		case key.Matches(msg, m.keys.refreshGames):
			fmt.Println("Refreshing games...")
			return m, nil
		}
	}

	// This will also call our delegate's update function.
	newListModel, cmd := m.list.Update(msg)
	m.list = newListModel
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m ModelMenu) View() string {
	return appStyle.Render(m.list.View())
}

package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	bullet = "•"
)

type ModelMenu struct {
	term         string
	width        int
	height       int
	txtStyle     lipgloss.Style
	Id           int
	list         list.Model
	keys         *listKeyMap
	delegateKeys *delegateKeyMap
}

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

	txtStyle := renderer.NewStyle()

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
	gamesList.SetShowStatusBar(false)
	gamesList.Styles = listDefaultStyles(renderer)
	gamesList.Styles.Title = renderer.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"}).
		Padding(0, 1)
	gamesList.Help.Styles.ShortKey = renderer.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#ECFD65"})
	gamesList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listKeys.togglePagination,
			listKeys.createGame,
			listKeys.refreshGames,
		}
	}

	return ModelMenu{
		txtStyle:     txtStyle,
		list:         gamesList,
		keys:         listKeys,
		delegateKeys: delegateKeys,
	}
}

func listDefaultStyles(r *lipgloss.Renderer) (s list.Styles) {
	verySubduedColor := lipgloss.AdaptiveColor{Light: "#DDDADA", Dark: "#3C3C3C"}
	subduedColor := lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"}

	s.TitleBar = r.NewStyle().Padding(0, 0, 1, 2) //nolint:mnd

	s.Title = r.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	s.Spinner = r.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#8E8E8E", Dark: "#747373"})

	s.FilterPrompt = r.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#ECFD65"})

	s.FilterCursor = r.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#EE6FF8", Dark: "#EE6FF8"})

	s.DefaultFilterCharacterMatch = r.NewStyle().Underline(true)

	s.StatusBar = r.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"}).
		Padding(0, 0, 1, 2) //nolint:mnd

	s.StatusEmpty = r.NewStyle().Foreground(subduedColor)

	s.StatusBarActiveFilter = r.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"})

	s.StatusBarFilterCount = r.NewStyle().Foreground(verySubduedColor)

	s.NoItems = r.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#909090", Dark: "#626262"})

	s.ArabicPagination = r.NewStyle().Foreground(subduedColor)

	s.PaginationStyle = r.NewStyle().PaddingLeft(2) //nolint:mnd

	s.HelpStyle = r.NewStyle().Padding(1, 0, 0, 2) //nolint:mnd

	s.ActivePaginationDot = r.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#847A85", Dark: "#979797"}).
		SetString(bullet)

	s.InactivePaginationDot = r.NewStyle().
		Foreground(verySubduedColor).
		SetString(bullet)

	s.DividerDot = r.NewStyle().
		Foreground(verySubduedColor).
		SetString(" " + bullet + " ")

	return s
}

func (m ModelMenu) Init() tea.Cmd {
	return nil
}

func (m ModelMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := m.txtStyle.Padding(1, 2).GetFrameSize()
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
			statusMessageStyle := m.txtStyle.Foreground(lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#04B575"}).Render

			m.delegateKeys.join.SetEnabled(true)
			newItem := item{
				title:       "New Item",
				description: "This is a new item.",
			}
			insCmd := m.list.InsertItem(0, newItem)
			statusCmd := m.list.NewStatusMessage(statusMessageStyle("Created " + newItem.Title()))
			return m, tea.Batch(insCmd, statusCmd)

		case key.Matches(msg, m.keys.refreshGames):
			fmt.Println("Refreshing games...")
			return m, nil
		}

	case JoinMsg:
		fmt.Println("Joining game:", msg.GameId)
	}

	// This will also call our delegate's update function.
	newListModel, cmd := m.list.Update(msg)
	m.list = newListModel
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m ModelMenu) View() string {
	return m.txtStyle.Padding(1, 2).Render(m.list.View())
}

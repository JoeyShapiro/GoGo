package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/google/uuid"
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
	ModelCreate  ModelCreate
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
			key.WithHelp(" ", "create game"),
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
	delegate := newItemDelegate(delegateKeys, txtStyle)
	gamesList := list.New(items, delegate, 0, 0)
	gamesList.Title = "Current Games"
	gamesList.SetShowStatusBar(false)
	gamesList.Styles = listDefaultStyles(txtStyle)
	gamesList.Styles.Title = txtStyle.
		Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"}).
		Padding(0, 1)
	gamesList.Help.Styles.ShortKey = txtStyle.Foreground(lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#ECFD65"})
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

func (m ModelMenu) Init() tea.Cmd {
	return nil
}

func (m ModelMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, ok := msg.(CreateMsg)
	if m.ModelCreate.Id != "" && !ok {
		updatedModel, cmd := m.ModelCreate.Update(msg)
		m.ModelCreate = updatedModel.(ModelCreate)
		return m, cmd
	}

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
			m.delegateKeys.join.SetEnabled(true)

			m.ModelCreate = ModelCreate{Id: uuid.New().String()}

			return m, m.ModelCreate.Init()

		case key.Matches(msg, m.keys.refreshGames):
			items := []list.Item{}
			for _, game := range games {
				items = append(items, item{
					title:       game.Id,
					description: fmt.Sprintf("Players: %d; Turns: %d; Size: %d", game.Players, len(game.Moves), game.BoardSize),
				})
			}

			return m, m.list.SetItems(items)
		}

	case CreateMsg:
		m.ModelCreate.Id = ""
		events <- msg

	case JoinMsg:
		fmt.Println("Joining game:", msg.GameId)

		game, exists := games[msg.GameId]
		if !exists {
			log.Error("Game not found", "game_id", msg.GameId)
			m.list.NewStatusMessage(m.txtStyle.Render("Game not found"))
			return m, nil
		}

		var piece Cell
		switch game.Players {
		case 0:
			piece = White
		case 1:
			piece = Black
		default:
			// show status message
			m.list.NewStatusMessage(m.txtStyle.Render("Too many players connected"))
			return m, nil
		}

		mGame := ModelGame{
			txtStyle: m.txtStyle,
			term:     m.term,
			width:    m.width,
			height:   m.height,
			Player:   piece,
			Conn:     make(chan tea.Msg, 1),
			Id:       game.Players,
			GameId:   msg.GameId,
		}

		game.Players++
		game.PlayerConns = append(game.PlayerConns, &mGame.Conn)

		return mGame, nil
	}

	// This will also call our delegate's update function.
	newListModel, cmd := m.list.Update(msg)
	m.list = newListModel
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m ModelMenu) View() string {
	if m.ModelCreate.Id != "" {
		return m.ModelCreate.View()
	} else {
		return m.txtStyle.Padding(1, 2).Render(m.list.View())
	}
}

func listDefaultStyles(txtStyle lipgloss.Style) (s list.Styles) {
	verySubduedColor := lipgloss.AdaptiveColor{Light: "#DDDADA", Dark: "#3C3C3C"}
	subduedColor := lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"}

	s.TitleBar = txtStyle.Padding(0, 0, 1, 2) //nolint:mnd

	s.Title = txtStyle.
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	s.Spinner = txtStyle.
		Foreground(lipgloss.AdaptiveColor{Light: "#8E8E8E", Dark: "#747373"})

	s.FilterPrompt = txtStyle.
		Foreground(lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#ECFD65"})

	s.FilterCursor = txtStyle.
		Foreground(lipgloss.AdaptiveColor{Light: "#EE6FF8", Dark: "#EE6FF8"})

	s.DefaultFilterCharacterMatch = txtStyle.Underline(true)

	s.StatusBar = txtStyle.
		Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"}).
		Padding(0, 0, 1, 2) //nolint:mnd

	s.StatusEmpty = txtStyle.Foreground(subduedColor)

	s.StatusBarActiveFilter = txtStyle.
		Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"})

	s.StatusBarFilterCount = txtStyle.Foreground(verySubduedColor)

	s.NoItems = txtStyle.
		Foreground(lipgloss.AdaptiveColor{Light: "#909090", Dark: "#626262"})

	s.ArabicPagination = txtStyle.Foreground(subduedColor)

	s.PaginationStyle = txtStyle.PaddingLeft(2) //nolint:mnd

	s.HelpStyle = txtStyle.Padding(1, 0, 0, 2) //nolint:mnd

	s.ActivePaginationDot = txtStyle.
		Foreground(lipgloss.AdaptiveColor{Light: "#847A85", Dark: "#979797"}).
		SetString(bullet)

	s.InactivePaginationDot = txtStyle.
		Foreground(verySubduedColor).
		SetString(bullet)

	s.DividerDot = txtStyle.
		Foreground(verySubduedColor).
		SetString(" " + bullet + " ")

	return s
}

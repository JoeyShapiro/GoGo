package main

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

type ModelGame struct {
	term     string
	width    int
	height   int
	txtStyle lipgloss.Style
	Id       int
	GameId   string
	Player   Cell
	Conn     chan tea.Msg
}

func listenCmd(m ModelGame) tea.Cmd {
	return func() tea.Msg {
		msg := <-m.Conn
		return msg
	}
}

func (m ModelGame) Init() tea.Cmd {
	return nil
}

func (m ModelGame) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case SendMsg:
		return m, listenCmd(m)
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
	case tea.KeyMsg:
		game, ok := games[m.GameId]
		if !ok {
			log.Error("Game not found", "game_id", m.GameId)
			return m, tea.Quit
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "a", "left":
			if game.Player == m.Player && game.Cursor > 0 {
				game.Cursor--
			}
		case "d", "right":
			if game.Player == m.Player && game.Cursor < len(game.Board)-1 {
				game.Cursor++
			}
		case "w", "up":
			if game.Player == m.Player && game.Cursor-BOARD_SIZE >= 0 {
				game.Cursor -= BOARD_SIZE
			}
		case "s", "down":
			if game.Player == m.Player && game.Cursor+BOARD_SIZE < len(game.Board) {
				game.Cursor += BOARD_SIZE
			}
		case "tab": // tab to pass
			if m.Player == White {
				m.Player = Black
			} else {
				m.Player = White
			}
		case " ":
			if game.Player == m.Player && game.Cursor >= 0 && game.Cursor < len(game.Board) {
				game.Board[game.Cursor] = game.Player
				game.Moves = append(game.Moves, Move{
					Turn:   len(game.Moves),
					Player: game.Player,
					NRow:   game.Cursor / BOARD_SIZE,
					NCol:   game.Cursor % BOARD_SIZE,
					Ctime:  uint64(time.Now().UTC().Unix()),
				})

				game.Last = game.Cursor
				if game.Player == White {
					game.Player = Black
				} else {
					game.Player = White
				}
			}
		}

		game.Conn <- SendMsg{Id: m.Id}
	}

	return m, listenCmd(m)
}

func (m ModelGame) View() string {
	game, ok := games[m.GameId]
	if !ok {
		log.Error("Game not found", "game_id", m.GameId)
		return "Game not found"
	}

	var b strings.Builder
	background := m.txtStyle.Background(lipgloss.Color("#af875f"))
	empty := background.Foreground(lipgloss.Color("#000000")).Render("┼")
	white := background.Foreground(lipgloss.Color("#ffffff")).Render("●")
	black := background.Foreground(lipgloss.Color("#000000")).Render("●")
	// last peice and currently selected will be cursor
	cursorBlack := background.Foreground(lipgloss.Color("#000000")).Render("○")
	cursorWhite := background.Foreground(lipgloss.Color("#ffffff")).Render("○")
	selected := m.txtStyle.Foreground(lipgloss.Color("#ff0000"))

	x := -1
	y := -1
	if game.Cursor > -1 {
		x = game.Cursor % BOARD_SIZE
		y = game.Cursor/BOARD_SIZE + 1
	}

	// top margin coordinates
	b.WriteRune(' ')
	for i := range BOARD_SIZE {
		margin := rune(i + 65)
		if x > -1 && i == x {
			b.WriteString(selected.Render(string(margin)))
		} else {
			b.WriteRune(margin)
		}
	}
	for i := range game.Board {
		// left margin coordinates
		if i%BOARD_SIZE == 0 {
			b.WriteRune('\n')
			row := i/BOARD_SIZE + 1
			margin := rune(row + 48)
			if y > -1 && row == y {
				b.WriteString(selected.Render(string(margin)))
			} else {
				b.WriteRune(margin)
			}
		}

		if i == game.Cursor {
			switch game.Player {
			case White:
				b.WriteString(cursorWhite)
			case Black:
				b.WriteString(cursorBlack)
			}
		} else if i == game.Last {
			switch game.Board[i] {
			case White:
				b.WriteString(cursorWhite)
			case Black:
				b.WriteString(cursorBlack)
			}
		} else {
			switch game.Board[i] {
			case Empty:
				b.WriteString(empty)
			case White:
				b.WriteString(white)
			case Black:
				b.WriteString(black)
			default:
				b.WriteString(" ")
			}
		}

		// right margin coordinates
		if i%BOARD_SIZE == BOARD_SIZE-1 {
			row := i/BOARD_SIZE + 1
			margin := rune(row + 48)
			if y > -1 && row == y {
				b.WriteString(selected.Render(string(margin)))
			} else {
				b.WriteRune(margin)
			}
		}
	}
	b.WriteRune(' ')
	b.WriteRune('\n')
	// bottom margin coordinates
	b.WriteRune(' ')
	for i := range BOARD_SIZE {
		margin := rune(i + 65)
		if x > -1 && i == x {
			b.WriteString(selected.Render(string(margin)))
		} else {
			b.WriteRune(margin)
		}
	}

	return b.String()
}

package main

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"

	_ "github.com/mattn/go-sqlite3"
)

const (
	host = "0.0.0.0"
	port = "23234"
)

var (
	game = NewGame("1")
)

//go:embed gogo.sql
var gogodotsql string

func main() {
	// Open database
	db, err := sql.Open("sqlite3", "./gogo.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := initdb(db); err != nil {
		log.Fatal("Failed to initialize database", "error", err)
	}

	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithMiddleware(
			bubbletea.Middleware(teaHandler),
			activeterm.Middleware(), // Bubble Tea apps usually require a PTY.
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Error("Could not start server", "error", err)
	}

	go func() {
		for {
			msg := <-game.Conn
			switch msg := msg.(type) {
			case SendMsg:
				for i := range game.Players {
					if i != msg.Id {
						*game.PlayerConns[i] <- SendMsg{Id: i}
					}
				}
			default:
				log.Warn("Unknown message type", "msg", msg)
			}
		}
	}()

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting SSH server", "host", host, "port", port)
	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not stop server", "error", err)
	}
}

func initdb(db *sql.DB) error {
	_, err := db.Exec(gogodotsql)
	return err
}

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	// This should never fail, as we are using the activeterm middleware.
	pty, _, _ := s.Pty()

	renderer := bubbletea.MakeRenderer(s)
	txtStyle := renderer.NewStyle()

	var piece Cell
	switch game.Players {
	case 0:
		piece = White
	case 1:
		piece = Black
	default:
		log.Error("Too many players connected", "players", game.Players)
		return nil, []tea.ProgramOption{tea.WithAltScreen()}
	}

	m := model{
		txtStyle: txtStyle,
		term:     pty.Term,
		width:    pty.Window.Width,
		height:   pty.Window.Height,
		Player:   piece,
		Conn:     make(chan tea.Msg, 1),
		Id:       game.Players,
	}

	game.Players++
	game.PlayerConns = append(game.PlayerConns, &m.Conn)

	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

// Just a generic tea.Model to demo terminal information of ssh.
type model struct {
	term     string
	width    int
	height   int
	txtStyle lipgloss.Style
	Id       int
	GameId   string
	Player   Cell
	Conn     chan tea.Msg
}

func listenCmd(m model) tea.Cmd {
	return func() tea.Msg {
		msg := <-m.Conn
		return msg
	}
}

type Game struct {
	Id            string
	Board         []Cell
	Cursor        int
	Last          int
	Player        Cell
	WhiteCaptures int
	BlackCaptures int
	Players       int
	Conn          chan tea.Msg
	PlayerConns   []*chan tea.Msg
}

func NewGame(id string) Game {
	return Game{
		Id:            id,
		Board:         make([]Cell, BOARD_SIZE*BOARD_SIZE),
		Cursor:        -1,
		Last:          -1,
		Player:        White,
		WhiteCaptures: 0,
		BlackCaptures: 0,
		Players:       0,
		Conn:          make(chan tea.Msg, 3),
	}
}

const BOARD_SIZE = 9 // Go board is 19x19
// 13 9

type Cell int

const (
	Empty Cell = iota
	White
	Black
)

type SendMsg struct {
	Id int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case SendMsg:
		return m, listenCmd(m)
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
	case tea.KeyMsg:
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
				game.Last = game.Cursor
				if game.Player == White {
					game.Player = Black
				} else {
					game.Player = White
				}
			}
		}
	}

	game.Conn <- SendMsg{Id: m.Id}

	return m, listenCmd(m)
}

func (m model) View() string {
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

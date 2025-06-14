package main

// An example Bubble Tea server. This will put an ssh session into alt screen
// and continually print up to date terminal information.

import (
	"context"
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
)

const (
	host = "0.0.0.0"
	port = "23234"
)

var (
	game = NewGame("1")
)

func main() {
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

// You can wire any Bubble Tea model up to the middleware with a function that
// handles the incoming ssh.Session. Here we just grab the terminal info and
// pass it to the new model. You can also return tea.ProgramOptions (such as
// tea.WithAltScreen) on a session by session basis.
func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	// This should never fail, as we are using the activeterm middleware.
	pty, _, _ := s.Pty()

	// When running a Bubble Tea app over SSH, you shouldn't use the default
	// lipgloss.NewStyle function.
	// That function will use the color profile from the os.Stdin, which is the
	// server, not the client.
	// We provide a MakeRenderer function in the bubbletea middleware package,
	// so you can easily get the correct renderer for the current session, and
	// use it to create the styles.
	// The recommended way to use these styles is to then pass them down to
	// your Bubble Tea model.
	renderer := bubbletea.MakeRenderer(s)
	txtStyle := renderer.NewStyle().Foreground(lipgloss.Color("10"))
	quitStyle := renderer.NewStyle().Foreground(lipgloss.Color("8"))

	bg := "light"
	if renderer.HasDarkBackground() {
		bg = "dark"
	}

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
		term:      pty.Term,
		profile:   renderer.ColorProfile().Name(),
		width:     pty.Window.Width,
		height:    pty.Window.Height,
		bg:        bg,
		txtStyle:  txtStyle,
		quitStyle: quitStyle,
		Player:    piece,
		Conn:      make(chan tea.Msg, 1),
		Id:        game.Players,
	}

	game.Players++
	game.Conns = append(game.Conns, &m.Conn)

	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

// Just a generic tea.Model to demo terminal information of ssh.
type model struct {
	term      string
	profile   string
	width     int
	height    int
	bg        string
	txtStyle  lipgloss.Style
	quitStyle lipgloss.Style
	GameId    string
	Player    Cell
	Conn      chan tea.Msg
	Id        int
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
	Conns         []*chan tea.Msg
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
	}
}

// could maybe have a history of all moves.
// but i would need to compute every turn, and check for captures, etc.
// and i would need the board on the frontend anyway
// only need to compute the next turn, but still would need the board
// doesnt save me, and might use a lot of memory
// just dump it in the db

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

	// send message to the channel
	for i, conn := range game.Conns {
		if i == m.Id {
			continue
		}

		*conn <- SendMsg{Id: m.Id}
	}

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

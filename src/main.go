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
	host = "localhost"
	port = "23234"
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

	m := model{
		term:      pty.Term,
		profile:   renderer.ColorProfile().Name(),
		width:     pty.Window.Width,
		height:    pty.Window.Height,
		bg:        bg,
		txtStyle:  txtStyle,
		quitStyle: quitStyle,
	}
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
	Board     []Cell
	Cursor    int
	Last      int
	Player    Cell
}

const BOARD_SIZE = 9 // Go board is 19x19
// 13 9

type Cell int

const (
	Empty Cell = iota
	White
	Black
)

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		m.Board = make([]Cell, BOARD_SIZE*BOARD_SIZE)

		// // make some random cells
		// for i := 0; i < len(m.Board); i++ {
		// 	if i%2 == 0 {
		// 		m.Board[i] = White
		// 	} else if i%3 == 0 {
		// 		m.Board[i] = Black
		// 	} else {
		// 		m.Board[i] = Empty
		// 	}
		// }

		m.Cursor = -1
		m.Last = -1
		m.Player = White

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "a", "left":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "d", "right":
			if m.Cursor < len(m.Board)-1 {
				m.Cursor++
			}
		case "w", "up":
			if m.Cursor-BOARD_SIZE >= 0 {
				m.Cursor -= BOARD_SIZE
			}
		case "s", "down":
			if m.Cursor+BOARD_SIZE < len(m.Board) {
				m.Cursor += BOARD_SIZE
			}
		}
	}
	return m, nil
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

	for i := range m.Board {
		if i > 0 && i%BOARD_SIZE == 0 && i < len(m.Board)-1 {
			b.WriteRune('\n')
		}

		if i == m.Cursor {
			switch m.Player {
			case White:
				b.WriteString(cursorWhite)
			case Black:
				b.WriteString(cursorBlack)
			}
		} else if i == m.Last {
			switch m.Board[i] {
			case White:
				b.WriteString(cursorWhite)
			case Black:
				b.WriteString(cursorBlack)
			}
		} else {
			switch m.Board[i] {
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
	}

	return b.String()
}

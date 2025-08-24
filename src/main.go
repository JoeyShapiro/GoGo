package main

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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

// TODO handle starting player
// TODO could add custom colors, but just use color scheme instead. best idea
// TODO auto update game list

var (
	games  map[string]*Game
	events chan tea.Msg
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

	games = make(map[string]*Game)
	events = make(chan tea.Msg, 10)

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
			// handle server events (non-blocking)
			select {
			case msg := <-events:
				switch msg := msg.(type) {
				case CreateMsg:
					log.Info("Creating game:", "id", msg.GameId)
					newGame := NewGame(msg.GameId, db)
					games[msg.GameId] = &newGame
				}
			default:
				// No server events, continue
			}

			// handle game events (non-blocking)
			for _, game := range games {
				select {
				case msg := <-game.Conn:
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
				default:
					// No message for this game, continue to next game
				}
			}

			time.Sleep(100 * time.Millisecond)
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
	// pty, _, _ := s.Pty()

	renderer := bubbletea.MakeRenderer(s)
	m := newModel(renderer)

	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

type Game struct {
	Id            string
	Board         []Cell
	BoardSize     int
	Cursor        int
	Last          int
	Player        Cell
	WhiteCaptures int
	BlackCaptures int
	Players       int
	Conn          chan tea.Msg
	PlayerConns   []*chan tea.Msg
	Moves         []Move
}

type Move struct {
	Turn   int
	Player Cell
	NRow   int
	NCol   int
	Ctime  uint64
}

func NewGame(id string, db *sql.DB) Game {
	_, err := db.Exec("INSERT INTO games (id, bsize, white, black, creation) VALUES (?, ?, ?, ?, ?)",
		id, BOARD_SIZE, "White", "Black", time.Now().UTC().Unix())
	if err != nil {
		log.Error("Failed to create game in database", "error", err)
		return Game{}
	}

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

func EndGame(id string, db *sql.DB) error {
	game, ok := games[id]
	if !ok {
		return errors.New("game not found")
	}

	for _, move := range game.Moves {
		_, err := db.Exec("INSERT INTO moves (game_id, turn, player, nrow, ncol, ctime) VALUES (?, ?, ?, ?, ?, ?)",
			id, move.Turn, move.Player, move.NRow, move.NCol, move.Ctime)
		if err != nil {
			return err
		}
	}

	game.Conn <- EndMsg{
		GameId: id,
		Winner: game.Player,
	}

	_, err := db.Exec("UPDATE games SET winner = ?, ended = ? WHERE id = ?", game.Player, time.Now().UTC().Unix(), id)
	if err != nil {
		return err
	}

	log.Info("Game ended", "game_id", id, "winner", game.Player)

	return nil
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

type EndMsg struct {
	GameId string
	Winner Cell
}

type JoinMsg struct {
	GameId string
}

type CreateMsg struct {
	GameId    string
	AgainstAI bool
}

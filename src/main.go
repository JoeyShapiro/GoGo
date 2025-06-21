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

	"github.com/google/uuid"

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

// TODO add more games some way, then interact with them

var (
	games map[string]*Game
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

	uuid := uuid.New().String()
	game := NewGame(uuid, db)
	games[uuid] = &game

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

	gameId := uuid.New().String()
	game, exists := games[gameId]
	if !exists {
		log.Error("Game not found", "game_id", gameId)
		return nil, []tea.ProgramOption{tea.WithAltScreen()}
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

	m := ModelGame{
		txtStyle: txtStyle,
		term:     pty.Term,
		width:    pty.Window.Width,
		height:   pty.Window.Height,
		Player:   piece,
		Conn:     make(chan tea.Msg, 1),
		Id:       game.Players,
		GameId:   gameId,
	}

	game.Players++
	game.PlayerConns = append(game.PlayerConns, &m.Conn)

	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

type ModelMenu struct {
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

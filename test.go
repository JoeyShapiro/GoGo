package main

// A simple example demonstrating how to draw and animate on a cellular grid.
// Note that the cellbuffer implementation in this example does not support
// double-width runes.

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"
)

const (
	fps       = 60
	frequency = 7.5
	damping   = 0.15
	asterisk  = "*"
)

type cellbuffer struct {
	cells  []string
	stride int
}

func (c *cellbuffer) init(w, h int) {
	if w == 0 {
		return
	}
	c.stride = w
	c.cells = make([]string, w*h)
	c.wipe()
}

func (c cellbuffer) set(x, y int) {
	i := y*c.stride + x
	if i > len(c.cells)-1 || x < 0 || y < 0 || x >= c.width() || y >= c.height() {
		return
	}
	c.cells[i] = asterisk
}

func (c *cellbuffer) wipe() {
	for i := range c.cells {
		c.cells[i] = " "
	}
}

func (c cellbuffer) width() int {
	return c.stride
}

func (c cellbuffer) height() int {
	h := len(c.cells) / c.stride
	if len(c.cells)%c.stride != 0 {
		h++
	}
	return h
}

func (c cellbuffer) ready() bool {
	return len(c.cells) > 0
}

func (c cellbuffer) String() string {
	var b strings.Builder
	for i := 0; i < len(c.cells); i++ {
		if i > 0 && i%c.stride == 0 && i < len(c.cells)-1 {
			b.WriteRune('\n')
		}
		b.WriteString(c.cells[i])
	}
	return b.String()
}

type frameMsg struct{}

func animate() tea.Cmd {
	return tea.Tick(time.Second/fps, func(_ time.Time) tea.Msg {
		return frameMsg{}
	})
}

type model struct {
	cells                cellbuffer
	spring               harmonica.Spring
	targetX, targetY     float64
	x, y                 float64
	xVelocity, yVelocity float64
}

func (m model) Init() tea.Cmd {
	return animate()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit
	case tea.WindowSizeMsg:
		if !m.cells.ready() {
			m.targetX, m.targetY = float64(msg.Width)/2, float64(msg.Height)/2
		}
		m.cells.init(msg.Width, msg.Height)
		return m, nil
	case tea.MouseMsg:
		if !m.cells.ready() {
			return m, nil
		}
		m.targetX, m.targetY = float64(msg.X), float64(msg.Y)
		return m, nil

	case frameMsg:
		if !m.cells.ready() {
			return m, nil
		}

		m.cells.wipe()
		m.x, m.xVelocity = m.spring.Update(m.x, m.xVelocity, m.targetX)
		m.y, m.yVelocity = m.spring.Update(m.y, m.yVelocity, m.targetY)
		drawEllipse(&m.cells, m.x, m.y, 16, 8)
		return m, animate()
	default:
		return m, nil
	}
}

func (m model) View() string {
	return m.cells.String()
}

func main() {
	m := model{
		spring: harmonica.NewSpring(harmonica.FPS(fps), frequency, damping),
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Println("Uh oh:", err)
		os.Exit(1)
	}
}

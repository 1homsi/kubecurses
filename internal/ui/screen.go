package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

// Screen wraps a tcell.Screen with convenience helpers.
type Screen struct {
	tcell.Screen
}

// NewScreen initialises and returns a new Screen.
func NewScreen() (*Screen, error) {
	s, err := tcell.NewScreen()
	if err != nil {
		return nil, fmt.Errorf("tcell.NewScreen: %w", err)
	}
	if err := s.Init(); err != nil {
		return nil, fmt.Errorf("screen.Init: %w", err)
	}
	s.SetStyle(StyleDefault)
	s.EnableMouse()
	s.Clear()
	return &Screen{s}, nil
}

// DrawText renders a string at (x, y) using the given style.
// Characters beyond the screen width are silently truncated.
func (s *Screen) DrawText(x, y int, style tcell.Style, text string) {
	w, h := s.Size()
	for i, ch := range text {
		col := x + i
		if col >= w || y >= h || y < 0 {
			break
		}
		s.SetContent(col, y, ch, nil, style)
	}
}

// DrawTextTrunc renders text truncated to maxWidth runes.
func (s *Screen) DrawTextTrunc(x, y, maxWidth int, style tcell.Style, text string) {
	runes := []rune(text)
	if len(runes) > maxWidth {
		runes = runes[:maxWidth]
	}
	s.DrawText(x, y, style, string(runes))
}

// FillRect fills a rectangle with the given rune and style.
func (s *Screen) FillRect(r Rect, ch rune, style tcell.Style) {
	w, h := s.Size()
	for row := r.Y; row < r.Y+r.H && row < h; row++ {
		for col := r.X; col < r.X+r.W && col < w; col++ {
			s.SetContent(col, row, ch, nil, style)
		}
	}
}

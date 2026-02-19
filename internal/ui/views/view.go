// Package views contains per-resource view implementations.
package views

import (
	"github.com/1homsi/kubecurses/internal/model"
	"github.com/1homsi/kubecurses/internal/ui"
)

// View is implemented by each resource-specific view.
type View interface {
	// Draw renders the view into the given rectangle.
	Draw(s *ui.Screen, r ui.Rect, state *model.AppState)
}

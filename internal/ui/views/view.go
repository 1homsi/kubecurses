// Package views contains per-resource view implementations.
package views

import (
	"github.com/1homsi/kubecurses/internal/model"
)

// View is implemented by each resource-specific view.
type View interface {
	// Render returns the view content as a string for the given dimensions.
	Render(width, height int, state *model.AppState) string
}

// podIcon returns a status icon for a pod, falling back to a text label when
// noIcons is true. Keeps the column widths consistent by padding text labels.
func podIcon(status string, noIcons bool) string {
	if noIcons {
		switch status {
		case "Running":
			return "[RUN]"
		case "Pending":
			return "[PND]"
		case "Terminating":
			return "[TRM]"
		default:
			return "[ERR]"
		}
	}
	switch status {
	case "Running":
		return "●"
	case "Pending":
		return "◑"
	case "Terminating":
		return "◌"
	default:
		return "✖"
	}
}

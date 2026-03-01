package views

import (
	"fmt"
	"time"
)

// formatDuration formats a duration into a human-readable age string
// similar to kubectl output (e.g. "5d", "3h", "10m", "45s").
func formatCPU(m int64) string {
	if m == 0 {
		return "0"
	}
	if m >= 1000 {
		return fmt.Sprintf("%.1f", float64(m)/1000.0)
	}
	return fmt.Sprintf("%dm", m)
}

func formatMem(mi int64) string {
	if mi == 0 {
		return "0"
	}
	if mi >= 1024 {
		return fmt.Sprintf("%dGi", mi/1024)
	}
	return fmt.Sprintf("%dMi", mi)
}

func formatDuration(d time.Duration) string {
	d = d.Truncate(time.Second)
	if d < 0 {
		d = 0
	}
	days := int(d.Hours()) / 24
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	hours := int(d.Hours())
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	minutes := int(d.Minutes())
	if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}

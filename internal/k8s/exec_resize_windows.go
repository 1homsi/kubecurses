//go:build windows

package k8s

import "k8s.io/client-go/tools/remotecommand"

// TermSizeQueue is a no-op on Windows — SIGWINCH-based terminal resize events
// are not supported. The remote PTY will use the size reported at session open.
type TermSizeQueue struct{}

// NewTermSizeQueue returns a no-op TermSizeQueue.
func NewTermSizeQueue() *TermSizeQueue { return &TermSizeQueue{} }

// Next always returns nil, signalling no resize events.
func (q *TermSizeQueue) Next() *remotecommand.TerminalSize { return nil }

// Stop is a no-op.
func (q *TermSizeQueue) Stop() {}

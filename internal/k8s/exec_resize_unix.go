//go:build !windows

package k8s

import (
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"

	"k8s.io/client-go/tools/remotecommand"
)

// TermSizeQueue implements remotecommand.TerminalSizeQueue by forwarding
// SIGWINCH signals as TerminalSize events. It also sends the current size
// immediately on creation so the remote PTY is sized correctly from the start.
type TermSizeQueue struct {
	ch   chan remotecommand.TerminalSize
	done chan struct{}
}

// NewTermSizeQueue creates a TermSizeQueue and starts listening for SIGWINCH.
// Call Stop() when the exec session ends to release the signal handler and goroutine.
func NewTermSizeQueue() *TermSizeQueue {
	q := &TermSizeQueue{
		ch:   make(chan remotecommand.TerminalSize, 4),
		done: make(chan struct{}),
	}
	go q.listen()
	return q
}

func (q *TermSizeQueue) listen() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	defer signal.Stop(sigCh)

	// Send the current terminal size immediately so the remote PTY is sized
	// correctly before the user types anything.
	q.sendSize()

	for {
		select {
		case <-sigCh:
			q.sendSize()
		case <-q.done:
			return
		}
	}
}

func (q *TermSizeQueue) sendSize() {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return
	}
	select {
	case q.ch <- remotecommand.TerminalSize{Width: uint16(w), Height: uint16(h)}:
	case <-q.done:
	}
}

// Next blocks until the terminal is resized or Stop() is called.
// Returns nil when Stop() has been called, signalling no further events.
func (q *TermSizeQueue) Next() *remotecommand.TerminalSize {
	select {
	case size := <-q.ch:
		return &size
	case <-q.done:
		return nil
	}
}

// Stop signals the TermSizeQueue to cease forwarding resize events.
// It is safe to call Stop() multiple times.
func (q *TermSizeQueue) Stop() {
	select {
	case <-q.done:
		// already stopped
	default:
		close(q.done)
	}
}

package pty

import "io"

// ─── Interface ────────────────────────────────────────────────────────────────

// PTY is the cross-platform interface to a pseudo-terminal.
// Implementations live in pty_windows.go (ConPTY) and pty_unix.go (creack/pty).
type PTY interface {
	io.ReadWriter

	// Resize updates the terminal dimensions sent to the child process.
	Resize(cols, rows int) error

	// Wait blocks until the child process exits and returns its exit error.
	Wait() error

	// Close terminates the child process and releases all resources.
	Close() error
}

//go:build linux || darwin

package pty

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

// ─── Unix PTY ─────────────────────────────────────────────────────────────────

type unixPTY struct {
	ptmx *os.File
	cmd  *exec.Cmd
}

// New spawns a PTY-backed shell process.
// Uses the standard Unix PTY interface via creack/pty.
// Reads $SHELL from the environment, falls back to /bin/bash.
func New(cols, rows int) (PTY, error) {
	shell := resolveShell()

	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
	)

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})
	if err != nil {
		return nil, fmt.Errorf("pty: failed to start shell %q: %w", shell, err)
	}

	return &unixPTY{ptmx: ptmx, cmd: cmd}, nil
}

// ─── PTY interface ────────────────────────────────────────────────────────────

func (p *unixPTY) Read(b []byte) (int, error)  { return p.ptmx.Read(b) }
func (p *unixPTY) Write(b []byte) (int, error) { return p.ptmx.Write(b) }

func (p *unixPTY) Resize(cols, rows int) error {
	return pty.Setsize(p.ptmx, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})
}

func (p *unixPTY) Wait() error {
	return p.cmd.Wait()
}

func (p *unixPTY) Close() error {
	err := p.ptmx.Close()
	if p.cmd.Process != nil {
		p.cmd.Process.Kill() //nolint:errcheck
	}
	return err
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// DefaultShell returns the short shell name for display purposes (e.g. "zsh").
func DefaultShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return "bash"
	}
	for i := len(shell) - 1; i >= 0; i-- {
		if shell[i] == '/' {
			return shell[i+1:]
		}
	}
	return shell
}

func resolveShell() string {
	if s := os.Getenv("SHELL"); s != "" {
		return s
	}
	return "/bin/bash"
}

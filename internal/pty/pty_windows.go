//go:build windows

package pty

import (
	"context"
	"fmt"

	"github.com/UserExistsError/conpty"
)

// ─── Windows PTY ──────────────────────────────────────────────────────────────

type windowsPTY struct {
	cpty *conpty.ConPty
}

// New spawns a PTY-backed process.
// Uses ConPTY which requires Windows 10 1809 or later.
// Tries PowerShell first, falls back to cmd.exe.
func New(cols, rows int) (PTY, error) {
	cpty, err := conpty.Start(
		`powershell.exe -NoLogo`,
		conpty.ConPtyDimensions(cols, rows),
	)
	if err != nil {
		cpty, err = conpty.Start(
			`cmd.exe`,
			conpty.ConPtyDimensions(cols, rows),
		)
		if err != nil {
			return nil, fmt.Errorf("pty: failed to start shell: %w", err)
		}
	}
	return &windowsPTY{cpty: cpty}, nil
}

// ─── PTY interface ────────────────────────────────────────────────────────────

func (p *windowsPTY) Read(b []byte) (int, error)  { return p.cpty.Read(b) }
func (p *windowsPTY) Write(b []byte) (int, error) { return p.cpty.Write(b) }
func (p *windowsPTY) Close() error                { return p.cpty.Close() }

func (p *windowsPTY) Resize(cols, rows int) error {
	return p.cpty.Resize(cols, rows)
}

func (p *windowsPTY) Wait() error {
	_, err := p.cpty.Wait(context.Background())
	return err
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// DefaultShell returns the short shell name for display purposes.
func DefaultShell() string { return "powershell" }

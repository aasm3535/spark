package ui

import (
	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/widget"
	sparkpty "yutug.lol/spark/internal/pty"
	"yutug.lol/spark/internal/terminal"
)

// Tab represents a single terminal session.
type Tab struct {
	term           *terminal.Terminal
	pty            sparkpty.PTY
	scrollBar      widget.Scrollbar
	scrollFraction float32
	closed         bool

	btnClick widget.Clickable
	btnClose widget.Clickable
}

// newTab creates a new PTY session and appends it to the window.
func (win *Window) newTab() error {
	p, err := sparkpty.New(terminal.DefaultCols, terminal.DefaultRows)
	if err != nil {
		return err
	}
	term := terminal.New(terminal.DefaultCols, terminal.DefaultRows, win.w)

	tab := &Tab{
		term: term,
		pty:  p,
	}
	win.tabs = append(win.tabs, tab)
	win.activeTab = len(win.tabs) - 1

	// reader goroutine: PTY -> terminal buffer
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := p.Read(buf)
			if n > 0 {
				term.Write(buf[:n]) //nolint:errcheck
			}
			if err != nil {
				tab.closed = true
				win.w.Invalidate()
				return
			}
		}
	}()

	return nil
}

// closeTab closes the tab at idx and cleans up its PTY.
func (win *Window) closeTab(idx int) {
	if idx < 0 || idx >= len(win.tabs) {
		return
	}
	win.tabs[idx].pty.Close()
	win.tabs = append(win.tabs[:idx], win.tabs[idx+1:]...)

	if len(win.tabs) == 0 {
		win.w.Perform(system.ActionClose)
		return
	}
	if win.activeTab >= len(win.tabs) {
		win.activeTab = len(win.tabs) - 1
	}
	win.w.Invalidate()
}

// activeTab returns the currently active Tab.
func (win *Window) active() *Tab {
	if len(win.tabs) == 0 {
		return nil
	}
	return win.tabs[win.activeTab]
}

// cleanup closes all open PTY sessions — called before exit.
func (win *Window) cleanup() {
	for _, t := range win.tabs {
		if t.pty != nil {
			t.pty.Close()
		}
	}
}

// ensure app is used
var _ *app.Window

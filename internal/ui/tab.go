package ui

import (
	"gioui.org/io/system"
	"gioui.org/widget"
	sparkpty "yutug.lol/spark/internal/pty"
	"yutug.lol/spark/internal/terminal"
	"yutug.lol/spark/internal/ui/components"
)

// Tab represents a single terminal session with all its associated state.
type Tab struct {
	term           *terminal.Terminal
	pty            sparkpty.PTY
	scrollBar      widget.Scrollbar
	scrollFraction float32
	closed         bool

	// TabState holds the clickable widgets used by TabBar.
	State components.TabState
}

// newTab spawns a new PTY + terminal and appends it to the window.
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

	// Feed PTY output into the terminal buffer.
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

// closeTab closes the PTY at idx and removes the tab from the list.
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

// active returns the currently visible Tab, or nil when there are none.
func (win *Window) active() *Tab {
	if len(win.tabs) == 0 {
		return nil
	}
	return win.tabs[win.activeTab]
}

// cleanup closes every open PTY — called on window exit.
func (win *Window) cleanup() {
	for _, t := range win.tabs {
		if t.pty != nil {
			t.pty.Close()
		}
	}
}

package ui

import (
	"os"
	"strings"
	"time"

	"gioui.org/io/system"
	"gioui.org/widget"
	"github.com/shirou/gopsutil/v3/process"
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

	// SbWidth tracks the animating width of the vertical scrollbar.
	SbWidth float32

	// MouseX tracks the current mouse X position for proximity detection.
	MouseX int

	// AI Agent detection state
	activeAgent      string
	activeSubprocess string
	lastProcessCheck time.Time
}

// newTab spawns a new PTY + terminal and appends it to the window.
func (win *Window) newTab() error {
	p, err := sparkpty.New(terminal.DefaultCols, terminal.DefaultRows)
	if err != nil {
		return err
	}
	term := terminal.New(terminal.DefaultCols, terminal.DefaultRows, win.w)

	// Wire PTY writer so terminal can send responses back
	term.SetPTYWriter(func(b []byte) {
		p.Write(b) //nolint:errcheck
	})

	tab := &Tab{
		term: term,
		pty:  p,
	}
	win.tabs = append(win.tabs, tab)
	win.activeTab = len(win.tabs) - 1

	// Set initial working directory from current OS dir
	if cwd, err := os.Getwd(); err == nil {
		term.SetWorkingDir(cwd)
	}

	// Background goroutine: poll shell process CWD every 2 seconds
	// to track cd changes without injecting any visible script.
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if tab.closed {
				return
			}
			pid := p.Pid()
			if pid == 0 {
				continue
			}

			// Try the PTY process itself
			cwd := pollCwd(int32(pid))
			if cwd != "" {
				term.SetWorkingDir(cwd)
			}
		}
	}()

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

// CheckActiveAgent scans child processes of the shell to detect active AI agents.
// It caches both the active agent and the active subprocess names for 1.5 seconds to preserve performance.
func (t *Tab) CheckActiveAgent() string {
	if time.Since(t.lastProcessCheck) < 1500*time.Millisecond {
		return t.activeAgent
	}
	t.lastProcessCheck = time.Now()

	pid := t.pty.Pid()
	if pid == 0 {
		t.activeAgent = ""
		t.activeSubprocess = ""
		return ""
	}

	children, err := sparkpty.GetChildProcesses(pid)
	if err != nil {
		t.activeAgent = ""
		t.activeSubprocess = ""
		return ""
	}

	hasClaude := false
	hasOpencode := false
	hasPi := false

	for _, child := range children {
		name := strings.ToLower(child)
		if strings.Contains(name, "claude") {
			hasClaude = true
		} else if strings.Contains(name, "opencode") {
			hasOpencode = true
		} else if strings.Contains(name, "pi") {
			hasPi = true
		}
	}

	if hasClaude {
		t.activeAgent = "Claude Code"
	} else if hasOpencode {
		t.activeAgent = "Opencode"
	} else if hasPi {
		t.activeAgent = "Pi"
	} else {
		t.activeAgent = ""
	}

	// Calculate active subprocess from the same snapshot
	t.activeSubprocess = ""
	if t.activeAgent != "" {
		for i := len(children) - 1; i >= 0; i-- {
			name := strings.ToLower(children[i])
			if strings.Contains(name, "powershell") ||
				strings.Contains(name, "cmd.exe") ||
				strings.Contains(name, "bash") ||
				strings.Contains(name, "zsh") ||
				strings.Contains(name, "conhost") ||
				strings.Contains(name, "openconsole") ||
				strings.Contains(name, "node") ||
				strings.Contains(name, "npm") ||
				strings.Contains(name, "claude") ||
				strings.Contains(name, "opencode") ||
				strings.Contains(name, "pi") {
				continue
			}

			clean := children[i]
			clean = strings.TrimSuffix(clean, ".exe")
			t.activeSubprocess = clean
			break
		}
	}

	return t.activeAgent
}

// GetActiveSubprocess returns the cached name of the active leaf child process executing under the shell/agent.
func (t *Tab) GetActiveSubprocess(agent string) string {
	t.CheckActiveAgent() // Ensure the cache is updated if needed
	return t.activeSubprocess
}

// IsAgentWorking returns true if the agent is actively working/running.
func (t *Tab) IsAgentWorking(agent string) bool {
	if t.GetActiveSubprocess(agent) != "" {
		return true
	}
	return t.term.IsAgentWorking(agent)
}

// pollCwd tries to read the current working directory of a process
// by checking the process itself and its children.
func pollCwd(pid int32) string {
	proc, err := process.NewProcess(pid)
	if err != nil {
		return ""
	}

	// Try the process itself
	cwd, err := proc.Cwd()
	if err == nil && cwd != "" {
		return cwd
	}

	// Try child processes (shell is often a child of conpty)
	children, err := proc.Children()
	if err != nil {
		return ""
	}
	for _, child := range children {
		cwd, err := child.Cwd()
		if err == nil && cwd != "" {
			return cwd
		}
	}

	return ""
}

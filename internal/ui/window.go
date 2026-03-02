package ui

import (
	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/widget/material"
	"yutug.lol/spark/internal/config"
)

// Window is the top-level UI state.
type Window struct {
	w        *app.Window
	theme    *material.Theme
	titleBar TitleBar
	renderer Renderer

	tabs      []*Tab
	activeTab int

	inputTag struct{}
	focused  bool
}

// New creates the Window and spawns the initial tab.
func New(w *app.Window) (*Window, error) {
	cfg, _ := config.Load()
	th := NewTheme(cfg)

	win := &Window{
		w:     w,
		theme: th,
	}

	if err := win.newTab(); err != nil {
		return nil, err
	}

	return win, nil
}

// Layout is the root layout function called every FrameEvent.
func (win *Window) Layout(gtx layout.Context, w *app.Window) layout.Dimensions {
	// Clean up any tabs whose PTY has exited.
	for i := len(win.tabs) - 1; i >= 0; i-- {
		if win.tabs[i].closed {
			win.closeTab(i)
		}
	}

	if len(win.tabs) == 0 {
		return layout.Dimensions{}
	}

	win.handleEvents(gtx)

	paint.Fill(gtx.Ops, ColorBg)

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return win.titleBar.Layout(gtx, win.theme, w, "spark")
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return win.layoutTabBar(gtx)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return win.layoutTerminal(gtx)
		}),
	)
}

// ReadyForClose cleans up all PTY sessions before exit.
func (win *Window) ReadyForClose() {
	win.cleanup()
}

// ensure system is used
var _ = system.ActionClose

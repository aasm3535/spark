package ui

import (
	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/widget/material"
	"yutug.lol/spark/internal/config"
	"yutug.lol/spark/internal/ui/components"
	// config is also used for BindingManager
)

// Window is the top-level UI state.
type Window struct {
	w        *app.Window
	theme    *material.Theme
	bindings *config.BindingManager

	titleBar components.TitleBar
	tabBar   components.TabBar
	renderer components.Renderer
	cmdPal   components.CommandPalette
	search   components.SearchBar

	tabs      []*Tab
	activeTab int

	inputTag     struct{}
	focused      bool
	cmdActive    bool
	searchActive bool
}

// New creates the Window and spawns the initial tab.
func New(w *app.Window) (*Window, error) {
	cfg, _ := config.Load()
	th := components.NewTheme(cfg)

	win := &Window{
		w:        w,
		theme:    th,
		bindings: config.NewBindingManager(cfg),
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

	paint.Fill(gtx.Ops, components.ColorBg)

	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
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
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			dims, res := win.cmdPal.Layout(gtx, win.theme, win.cmdActive)
			if res.Closed || res.Submitted {
				win.cmdActive = false
				if res.Action != config.ActionNone {
					win.handleAction(res.Action)
				}
				gtx.Execute(key.FocusCmd{Tag: &win.inputTag})
				win.w.Invalidate()
			}

			// Layout Search bar
			_, sRes := win.search.Layout(gtx, win.theme, win.searchActive)
			if sRes.Closed {
				win.searchActive = false
				if active := win.active(); active != nil {
					active.term.SetSearch("")
				}
				gtx.Execute(key.FocusCmd{Tag: &win.inputTag})
				win.w.Invalidate()
			}
			if active := win.active(); active != nil {
				if sRes.QueryChanged {
					active.term.SetSearch(sRes.Query)
					win.w.Invalidate()
				}
				if sRes.Next {
					active.term.SearchNext()
					win.w.Invalidate()
				}
				if sRes.Prev {
					active.term.SearchPrev()
					win.w.Invalidate()
				}
			}

			return dims
		}),
	)
}

// ReadyForClose cleans up all PTY sessions before exit.
func (win *Window) ReadyForClose() {
	win.cleanup()
}

// ensure system is used
var _ = system.ActionClose

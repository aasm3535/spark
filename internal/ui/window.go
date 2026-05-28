package ui

import (
	"time"

	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"yutug.lol/spark/internal/config"
	"yutug.lol/spark/internal/ui/components"
)

// Window is the top-level UI state.
type Window struct {
	w        *app.Window
	theme    *material.Theme
	bindings *config.BindingManager
	config   *config.Config

	titleBar components.TitleBar
	sidebar  components.Sidebar
	tabBar   components.TabBar
	renderer components.Renderer
	cmdPal   components.CommandPalette
	search   components.SearchBar
	toast    components.Toast

	tabs      []*Tab
	activeTab int

	inputTag     struct{}
	focused       bool
	cmdActive     bool
	searchActive  bool
	sidebarActive bool
}

// New creates the Window and spawns the initial tab.
func New(w *app.Window) (*Window, error) {
	cfg, _ := config.Load()
	th := components.NewTheme(cfg)

	win := &Window{
		w:             w,
		theme:         th,
		bindings:      config.NewBindingManager(cfg),
		config:        cfg,
		sidebarActive: true,
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
					title := "Spark"
					if active := win.active(); active != nil {
						if tTitle := active.term.Title(); tTitle != "" {
							title = tTitle
						}
					}
					dims, res := win.titleBar.Layout(gtx, win.theme, w, title)
					if res.MenuClicked {
						win.sidebarActive = !win.sidebarActive
						win.w.Invalidate()
					}
					return dims
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if !win.sidebarActive {
								return layout.Dimensions{}
							}

							// Собираем заголовки вкладок
							titles := make([]string, len(win.tabs))
							for i, t := range win.tabs {
								titles[i] = t.term.Title()
							}

							// Синхронизируем вкладки в боковой панели
							for len(win.sidebar.Tabs) < len(win.tabs) {
								win.sidebar.Tabs = append(win.sidebar.Tabs, &components.TabState{})
							}
							win.sidebar.Tabs = win.sidebar.Tabs[:len(win.tabs)]
							for i, tab := range win.tabs {
								win.sidebar.Tabs[i] = &tab.State
							}

							dims, res := win.sidebar.Layout(gtx, win.theme, win.activeTab, titles)
							if res.NewTabClicked {
								win.newTab() //nolint:errcheck
							}
							if res.CmdPalClicked {
								win.cmdActive = !win.cmdActive
								if win.cmdActive {
									win.cmdPal.Editor.SetText("")
								}
								win.w.Invalidate()
							}
							if res.TabSwitchedTo != -1 {
								win.activeTab = res.TabSwitchedTo
								win.w.Invalidate()
							}
							if res.TabClosedIdx != -1 {
								win.tabs[res.TabClosedIdx].closed = true
								win.w.Invalidate()
							}
							return dims
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return win.layoutTerminal(gtx)
						}),
					)
				}),
			)
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			dims, res := win.cmdPal.Layout(gtx, win.theme, win.cmdActive)
			if res.Closed || res.Submitted {
				win.cmdActive = false
				if res.Action != config.ActionNone {
					win.handleAction(gtx, res.Action)
				}
				gtx.Execute(key.FocusCmd{Tag: &win.inputTag})
				win.w.Invalidate()
			}

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

			win.toast.Layout(gtx, win.theme)

			return dims
		}),
	)
}

// ReadyForClose cleans up all PTY sessions before exit.
func (win *Window) ReadyForClose() {
	win.cleanup()
}

// changeFontSize adjusts the terminal font size by delta points.
func (win *Window) changeFontSize(delta float32) {
	newSize := float32(win.theme.TextSize) + delta
	if newSize < 8 {
		newSize = 8
	}
	if newSize > 48 {
		newSize = 48
	}
	win.theme.TextSize = unit.Sp(newSize)
	win.w.Invalidate()
}

// ShowToast displays a temporary message at the bottom of the window.
func (win *Window) ShowToast(msg string) {
	win.toast.Show(msg, 2*time.Second)
	win.w.Invalidate()
	time.AfterFunc(2*time.Second, func() {
		win.w.Invalidate()
	})
}

// ensure system is used
var _ = system.ActionClose

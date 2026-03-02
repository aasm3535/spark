package ui

import (
	"fmt"
	"image"
	"image/color"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"yutug.lol/spark/internal/config"
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

// newTab creates a new PTY session and adds it to the window.
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

// closeTab forcefully closes a tab by index.
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

// ─── Layout ───────────────────────────────────────────────────────────────────

// Layout is the root layout function called every FrameEvent.
func (win *Window) Layout(gtx layout.Context, w *app.Window) layout.Dimensions {
	// Clean up closed tabs
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

func (win *Window) layoutTabBar(gtx layout.Context) layout.Dimensions {
	if len(win.tabs) <= 1 {
		return layout.Dimensions{}
	}

	var children []layout.FlexChild
	for i, tab := range win.tabs {
		i := i
		tab := tab
		isActive := i == win.activeTab

		// Handle clicks
		if tab.btnClick.Clicked(gtx) {
			win.activeTab = i
			win.w.Invalidate()
		}
		if tab.btnClose.Clicked(gtx) {
			tab.closed = true
			win.w.Invalidate()
		}

		children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			bg := ColorTabInactiveBg
			if isActive {
				bg = ColorTabActiveBg
			} else if tab.btnClick.Hovered() {
				bg = ColorTabHoverBg
			}

			m := op.Record(gtx.Ops)

			dims := tab.btnClick.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X

				// Ширина правой кнопки: padding(4)*2 + size(16) + inset right(6) = 30dp
				closeW := gtx.Dp(unit.Dp(4 + 4 + 16 + 6))
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					// Пустой спейсер слева равный ширине крестика — чтобы текст был по центру всего таба
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Dimensions{Size: image.Pt(closeW, 0)}
					}),
					// Текст по центру
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								lbl := material.Label(win.theme, unit.Sp(12), fmt.Sprintf("Tab %d", i+1))
								lbl.Font = font.Font{
									Typeface: "Segoe UI, sans-serif",
									Weight:   font.Normal,
								}
								if isActive {
									lbl.Color = ColorText
								} else {
									lbl.Color = ColorTitleText
								}
								lbl.MaxLines = 1
								return lbl.Layout(gtx)
							})
						})
					}),
					// Крестик справа
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Right: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return tab.btnClose.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									size := gtx.Dp(unit.Dp(16))
									drawSize := float32(gtx.Dp(unit.Dp(8)))
									padding := (float32(size) - drawSize) / 2
									gtx.Constraints = layout.Exact(image.Pt(size, size))

									// Круглый фон при наведении на крестик
									if tab.btnClose.Hovered() {
										circleColor := color.NRGBA{R: 255, G: 255, B: 255, A: 30}
										paint.FillShape(gtx.Ops, circleColor, clip.Ellipse{
											Min: image.Pt(0, 0),
											Max: image.Pt(size, size),
										}.Op(gtx.Ops))
									}

									if tab.btnClick.Hovered() || tab.btnClose.Hovered() || isActive {
										colorX := ColorTitleText
										if tab.btnClose.Hovered() {
											colorX = ColorText
										}

										var p clip.Path
										p.Begin(gtx.Ops)
										p.MoveTo(f32.Pt(padding, padding))
										p.LineTo(f32.Pt(padding+drawSize, padding+drawSize))
										p.MoveTo(f32.Pt(padding+drawSize, padding))
										p.LineTo(f32.Pt(padding, padding+drawSize))

										paint.FillShape(gtx.Ops, colorX, clip.Stroke{Path: p.End(), Width: float32(gtx.Dp(unit.Dp(1)))}.Op())
									}

									return layout.Dimensions{Size: image.Pt(size, size)}
								})
							})
						})
					}),
				)
			})

			call := m.Stop()
			paint.FillShape(gtx.Ops, bg, clip.Rect{Max: dims.Size}.Op())
			call.Add(gtx.Ops)

			return dims
		}))

		// Small separator
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(gtx.Dp(unit.Dp(1)), 0)}
		}))
	}

	// Wrapper background for the tab bar
	m := op.Record(gtx.Ops)
	dims := layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
	call := m.Stop()

	paint.FillShape(gtx.Ops, ColorTitleBar, clip.Rect{Max: dims.Size}.Op())
	call.Add(gtx.Ops)

	return dims
}

func (win *Window) layoutTerminal(gtx layout.Context) layout.Dimensions {
	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()

	tag := &win.inputTag
	event.Op(gtx.Ops, tag)

	if !win.focused {
		gtx.Execute(key.FocusCmd{Tag: tag})
	}

	paint.Fill(gtx.Ops, ColorBg)

	if len(win.tabs) == 0 {
		return layout.Dimensions{}
	}
	active := win.tabs[win.activeTab]
	snap := active.term.Snapshot()

	if d := active.scrollBar.ScrollDistance(); d != 0 {
		total := snap.ScrollTotal + snap.Rows
		active.scrollFraction += d * float32(total)
		if active.scrollFraction >= 1 || active.scrollFraction <= -1 {
			delta := int(active.scrollFraction)
			active.scrollFraction -= float32(delta)
			active.term.Scroll(-delta)
			snap = active.term.Snapshot()
		}
	}

	var vStart, vEnd float32 = 0, 1
	if snap.ScrollTotal > 0 {
		total := float32(snap.ScrollTotal + snap.Rows)
		vEnd = 1.0 - float32(snap.ScrollOffset)/total
		vStart = vEnd - float32(snap.Rows)/total
	}

	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(6)).Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					dims := win.renderer.Layout(gtx, win.theme, snap)

					cols, rows := win.renderer.ColsRows(
						gtx.Constraints.Max.X-gtx.Dp(unit.Dp(12)),
						gtx.Constraints.Max.Y-gtx.Dp(unit.Dp(12)),
					)
					if cols != snap.Cols || rows != snap.Rows {
						active.term.Resize(cols, rows)
						active.pty.Resize(cols, rows) //nolint:errcheck
					}

					return dims
				},
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if snap.ScrollTotal == 0 {
				return layout.Dimensions{}
			}
			sb := material.Scrollbar(win.theme, &active.scrollBar)
			sb.Track.Color.A = 0
			sb.Track.MajorPadding = 0
			sb.Track.MinorPadding = 0
			sb.Indicator.MinorWidth = unit.Dp(4)
			sb.Indicator.CornerRadius = unit.Dp(2)
			sb.Indicator.Color.A = 50
			sb.Indicator.HoverColor.A = 150
			return sb.Layout(gtx, layout.Vertical, vStart, vEnd)
		}),
	)
}

// ─── Event handling ───────────────────────────────────────────────────────────

func (win *Window) handleEvents(gtx layout.Context) {
	tag := &win.inputTag

	filters := []event.Filter{
		pointer.Filter{
			Target:  tag,
			Kinds:   pointer.Scroll,
			ScrollY: pointer.ScrollRange{Min: -1000000, Max: 1000000},
		},
		key.FocusFilter{Target: tag},
		key.Filter{Focus: tag, Name: "", Optional: key.ModCtrl | key.ModShift | key.ModAlt},
		key.Filter{Focus: tag, Name: key.NameReturn},
		key.Filter{Focus: tag, Name: key.NameEnter},
		key.Filter{Focus: tag, Name: key.NameDeleteBackward},
		key.Filter{Focus: tag, Name: key.NameDeleteForward},
		key.Filter{Focus: tag, Name: key.NameUpArrow},
		key.Filter{Focus: tag, Name: key.NameUpArrow, Required: key.ModShift},
		key.Filter{Focus: tag, Name: key.NameDownArrow},
		key.Filter{Focus: tag, Name: key.NameDownArrow, Required: key.ModShift},
		key.Filter{Focus: tag, Name: key.NameLeftArrow},
		key.Filter{Focus: tag, Name: key.NameRightArrow},
		key.Filter{Focus: tag, Name: key.NameHome},
		key.Filter{Focus: tag, Name: key.NameEnd},
		key.Filter{Focus: tag, Name: key.NamePageUp},
		key.Filter{Focus: tag, Name: key.NamePageUp, Required: key.ModShift},
		key.Filter{Focus: tag, Name: key.NamePageUp, Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: key.NamePageDown},
		key.Filter{Focus: tag, Name: key.NamePageDown, Required: key.ModShift},
		key.Filter{Focus: tag, Name: key.NamePageDown, Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "T", Required: key.ModCtrl | key.ModShift},
		key.Filter{Focus: tag, Name: "W", Required: key.ModCtrl | key.ModShift},
		key.Filter{Focus: tag, Name: key.NameEscape},
		key.Filter{Focus: tag, Name: key.NameTab},
		key.Filter{Focus: tag, Name: key.NameTab, Required: key.ModShift},
		key.Filter{Focus: tag, Name: key.NameSpace},
		key.Filter{Focus: tag, Name: "A", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "B", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "C", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "D", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "E", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "F", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "G", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "H", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "I", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "J", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "K", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "L", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "M", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "N", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "O", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "P", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "Q", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "R", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "S", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "T", Required: key.ModCtrl, Optional: key.ModShift},
		key.Filter{Focus: tag, Name: "U", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "V", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "W", Required: key.ModCtrl, Optional: key.ModShift},
		key.Filter{Focus: tag, Name: "X", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "Y", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "Z", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "\\", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "]", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "[", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "-", Required: key.ModCtrl},
		key.Filter{Focus: tag, Name: "F1"},
		key.Filter{Focus: tag, Name: "F2"},
		key.Filter{Focus: tag, Name: "F3"},
		key.Filter{Focus: tag, Name: "F4"},
		key.Filter{Focus: tag, Name: "F5"},
		key.Filter{Focus: tag, Name: "F6"},
		key.Filter{Focus: tag, Name: "F7"},
		key.Filter{Focus: tag, Name: "F8"},
		key.Filter{Focus: tag, Name: "F9"},
		key.Filter{Focus: tag, Name: "F10"},
		key.Filter{Focus: tag, Name: "F11"},
		key.Filter{Focus: tag, Name: "F12"},
	}

	for {
		ev, ok := gtx.Event(filters...)
		if !ok {
			break
		}

		if len(win.tabs) == 0 {
			continue
		}
		active := win.tabs[win.activeTab]

		switch e := ev.(type) {
		case pointer.Event:
			if e.Kind == pointer.Scroll {
				if e.Scroll.Y > 0 {
					active.term.Scroll(-3)
				} else if e.Scroll.Y < 0 {
					active.term.Scroll(3)
				}
			}

		case key.FocusEvent:
			win.focused = e.Focus

		case key.Event:
			// Handle custom tab bindings first
			if e.State == key.Press {
				isCtrl := e.Modifiers.Contain(key.ModCtrl)
				isShift := e.Modifiers.Contain(key.ModShift)

				if isCtrl && isShift {
					if e.Name == "T" {
						win.newTab()
						continue
					}
					if e.Name == "W" {
						win.closeTab(win.activeTab)
						continue
					}
				}

				if isCtrl && !isShift {
					if e.Name == key.NamePageDown {
						win.activeTab = (win.activeTab + 1) % len(win.tabs)
						win.w.Invalidate()
						continue
					}
					if e.Name == key.NamePageUp {
						win.activeTab = (win.activeTab - 1 + len(win.tabs)) % len(win.tabs)
						win.w.Invalidate()
						continue
					}
				}
			}

			// Handle regular shift keybinds (scrolling)
			if e.Modifiers == key.ModShift {
				switch e.Name {
				case key.NameUpArrow:
					if e.State == key.Press {
						active.term.Scroll(1)
					}
					continue
				case key.NameDownArrow:
					if e.State == key.Press {
						active.term.Scroll(-1)
					}
					continue
				case key.NamePageUp:
					if e.State == key.Press {
						active.term.Scroll(10)
					}
					continue
				case key.NamePageDown:
					if e.State == key.Press {
						active.term.Scroll(-10)
					}
					continue
				}
			}

			b := terminal.KeyToBytes(e, active.term.AppCursorKeys())
			if len(b) > 0 {
				active.term.Scroll(-999999) // Reset scroll on typing
				active.pty.Write(b)         //nolint:errcheck
			}

		case key.EditEvent:
			if active.pty != nil && e.Text != " " {
				active.term.Scroll(-999999)      // Reset scroll on typing
				active.pty.Write([]byte(e.Text)) //nolint:errcheck
			}
		}
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// ReadyForClose cleans up all PTY sessions before exit.
func (win *Window) ReadyForClose() {
	for _, t := range win.tabs {
		if t.pty != nil {
			t.pty.Close()
		}
	}
}

// ensure image is used
var _ = image.Point{}

// suppress unused warning for op
var _ = op.Ops{}

// suppress unused warning for material
var _ *material.Theme

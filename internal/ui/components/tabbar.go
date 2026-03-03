package components

import (
	"fmt"
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// TabState holds the per-tab widget state (click + close buttons).
type TabState struct {
	BtnClick widget.Clickable
	BtnClose widget.Clickable
}

// TabBar renders the row of tabs above the terminal.
type TabBar struct {
	// Tabs holds one entry per open tab; the caller manages the slice.
	Tabs []*TabState
}

// TabBarEvent is returned by Layout to notify the caller of user actions.
type TabBarEvent int

const (
	TabBarNone    TabBarEvent = iota
	TabBarSwitchN             // caller should read SwitchedTo
	TabBarCloseN              // caller should read ClosedIdx
)

// TabBarResult is the output of a single Layout call.
type TabBarResult struct {
	Event      TabBarEvent
	SwitchedTo int
	ClosedIdx  int
}

// Layout draws the tab bar and returns any interaction that occurred.
func (tb *TabBar) Layout(
	gtx layout.Context,
	th *material.Theme,
	activeIdx int,
) (layout.Dimensions, TabBarResult) {
	if len(tb.Tabs) <= 1 {
		return layout.Dimensions{}, TabBarResult{}
	}

	var result TabBarResult

	var children []layout.FlexChild
	for i, tab := range tb.Tabs {
		i := i
		tab := tab
		isActive := i == activeIdx

		// Process click events and surface them to the caller.
		if tab.BtnClick.Clicked(gtx) {
			result = TabBarResult{Event: TabBarSwitchN, SwitchedTo: i}
		}
		if tab.BtnClose.Clicked(gtx) {
			result = TabBarResult{Event: TabBarCloseN, ClosedIdx: i}
		}

		children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			bg := ColorTabInactiveBg
			if isActive {
				bg = ColorTabActiveBg
			} else if tab.BtnClick.Hovered() {
				bg = ColorTabHoverBg
			}

			m := op.Record(gtx.Ops)
			dims := tab.BtnClick.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return layoutTabItem(gtx, th, tab, i, isActive)
			})
			call := m.Stop()

			paint.FillShape(gtx.Ops, bg, clip.Rect{Max: dims.Size}.Op())
			call.Add(gtx.Ops)

			return dims
		}))

		// Thin separator between tabs.
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(gtx.Dp(unit.Dp(1)), 0)}
		}))
	}

	// Record the flex into a macro so we can paint the background first.
	m := op.Record(gtx.Ops)
	dims := layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
	call := m.Stop()

	paint.FillShape(gtx.Ops, ColorTitleBar, clip.Rect{Max: dims.Size}.Op())
	call.Add(gtx.Ops)

	return dims, result
}

// ─── Tab item ─────────────────────────────────────────────────────────────────

// layoutTabItem draws the label (centred) and close button (right-pinned).
func layoutTabItem(
	gtx layout.Context,
	th *material.Theme,
	tab *TabState,
	idx int,
	isActive bool,
) layout.Dimensions {
	// closeW mirrors the close button so the label is truly centred.
	closeW := gtx.Dp(unit.Dp(4 + 4 + 16 + 6))

	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		// Left spacer = close button width.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(closeW, 0)}
		}),

		// Centred label.
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, unit.Sp(12), fmt.Sprintf("Tab %d", idx+1))
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

		// Close button.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layoutCloseButton(gtx, tab, isActive)
		}),
	)
}

// ─── Close button ─────────────────────────────────────────────────────────────

// layoutCloseButton draws the × icon with a circular hover highlight.
func layoutCloseButton(gtx layout.Context, tab *TabState, isActive bool) layout.Dimensions {
	return layout.Inset{Right: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return tab.BtnClose.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				size := gtx.Dp(unit.Dp(16))
				drawSize := float32(gtx.Dp(unit.Dp(8)))
				padding := (float32(size) - drawSize) / 2
				gtx.Constraints = layout.Exact(image.Pt(size, size))

				// Circular hover highlight.
				if tab.BtnClose.Hovered() {
					paint.FillShape(gtx.Ops,
						color.NRGBA{R: 255, G: 255, B: 255, A: 30},
						clip.Ellipse{Min: image.Pt(0, 0), Max: image.Pt(size, size)}.Op(gtx.Ops),
					)
				}

				// Draw × when active or hovered.
				if isActive || tab.BtnClick.Hovered() || tab.BtnClose.Hovered() {
					col := ColorTitleText
					if tab.BtnClose.Hovered() {
						col = ColorText
					}

					var p clip.Path
					p.Begin(gtx.Ops)
					p.MoveTo(f32.Pt(padding, padding))
					p.LineTo(f32.Pt(padding+drawSize, padding+drawSize))
					p.MoveTo(f32.Pt(padding+drawSize, padding))
					p.LineTo(f32.Pt(padding, padding+drawSize))

					paint.FillShape(gtx.Ops, col,
						clip.Stroke{Path: p.End(), Width: float32(gtx.Dp(unit.Dp(1)))}.Op(),
					)
				}

				return layout.Dimensions{Size: image.Pt(size, size)}
			})
		})
	})
}

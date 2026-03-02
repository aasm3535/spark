package ui

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
	"gioui.org/widget/material"
)

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
				return layoutTabItem(gtx, win.theme, tab, i, isActive)
			})
			call := m.Stop()

			paint.FillShape(gtx.Ops, bg, clip.Rect{Max: dims.Size}.Op())
			call.Add(gtx.Ops)

			return dims
		}))

		// Thin separator between tabs
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(gtx.Dp(unit.Dp(1)), 0)}
		}))
	}

	// Wrapper background for the whole tab bar
	m := op.Record(gtx.Ops)
	dims := layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
	call := m.Stop()

	paint.FillShape(gtx.Ops, ColorTitleBar, clip.Rect{Max: dims.Size}.Op())
	call.Add(gtx.Ops)

	return dims
}

// layoutTabItem draws the contents of a single tab: centred label + close button.
func layoutTabItem(gtx layout.Context, th *material.Theme, tab *Tab, idx int, isActive bool) layout.Dimensions {
	// closeW = UniformInset(4)*2 + size(16) + Right inset(6) = 30dp
	closeW := gtx.Dp(unit.Dp(4 + 4 + 16 + 6))

	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		// Left spacer mirrors close button width so the label is truly centred.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(closeW, 0)}
		}),

		// Centred label
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

		// Close button pinned to the right
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layoutCloseButton(gtx, tab, isActive)
		}),
	)
}

// layoutCloseButton draws the × icon with a circular hover highlight.
func layoutCloseButton(gtx layout.Context, tab *Tab, isActive bool) layout.Dimensions {
	return layout.Inset{Right: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return tab.btnClose.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				size := gtx.Dp(unit.Dp(16))
				drawSize := float32(gtx.Dp(unit.Dp(8)))
				padding := (float32(size) - drawSize) / 2
				gtx.Constraints = layout.Exact(image.Pt(size, size))

				// Circular hover highlight
				if tab.btnClose.Hovered() {
					paint.FillShape(gtx.Ops,
						color.NRGBA{R: 255, G: 255, B: 255, A: 30},
						clip.Ellipse{Min: image.Pt(0, 0), Max: image.Pt(size, size)}.Op(gtx.Ops),
					)
				}

				// Draw × only when the tab is active or hovered
				if isActive || tab.btnClick.Hovered() || tab.btnClose.Hovered() {
					col := ColorTitleText
					if tab.btnClose.Hovered() {
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

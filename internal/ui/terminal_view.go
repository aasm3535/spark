package ui

import (
	"image"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"yutug.lol/spark/internal/ui/components"
)

func (win *Window) layoutTerminal(gtx layout.Context) layout.Dimensions {
	defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()

	tag := &win.inputTag
	event.Op(gtx.Ops, tag)

	if !win.focused && !win.cmdActive && !win.searchActive {
		gtx.Execute(key.FocusCmd{Tag: tag})
	}

	paint.Fill(gtx.Ops, components.ColorBg)

	active := win.active()
	if active == nil {
		return layout.Dimensions{}
	}

	snap := active.term.Snapshot()

	// Handle scrollbar drag.
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

	pad := 5
	if win.config != nil {
		pad = win.config.Padding
	}
	padDp := unit.Dp(pad)
	var topPad unit.Dp
	if padDp > 3 {
		topPad = padDp - 3
	} else {
		topPad = 0
	}

	return layout.Stack{Alignment: layout.NW}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			inset := layout.Inset{
				Top:    topPad,
				Bottom: padDp,
				Left:   padDp,
				Right:  padDp,
			}
			return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				cols, rows := win.renderer.ColsRows(
					gtx.Constraints.Max.X,
					gtx.Constraints.Max.Y,
				)
				if cols != snap.Cols || rows != snap.Rows {
					active.term.Resize(cols, rows)
					active.pty.Resize(cols, rows) //nolint:errcheck
				}

				return layout.NW.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return win.renderer.Layout(gtx, win.theme, snap)
				})
			})
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			if snap.ScrollTotal == 0 {
				return layout.Dimensions{}
			}

			gtx.Constraints.Min.Y = gtx.Constraints.Max.Y

			// Calculate exact X position to place the scrollbar flush with the terminal text.
			windowWidth := gtx.Constraints.Max.X
			cw := win.renderer.CellWidth
			cols := snap.Cols
			padPx := gtx.Dp(padDp)
			textRightEdge := padPx + cols*cw

			baseTargetX := textRightEdge + gtx.Dp(unit.Dp(4))

			// Check if mouse is near the scrollbar area
			proximityZone := gtx.Dp(unit.Dp(30))
			mouseNearby := active.MouseX >= baseTargetX-proximityZone && active.MouseX <= baseTargetX+proximityZone

			hovered := active.scrollBar.IndicatorHovered() || active.scrollBar.TrackHovered() || active.scrollBar.Dragging()
			targetWidth := float32(5.0)
			if hovered {
				targetWidth = 10.0
			} else if mouseNearby {
				targetWidth = 8.0
			}

			if active.SbWidth == 0 {
				active.SbWidth = targetWidth
			}

			diff := targetWidth - active.SbWidth
			if diff < -0.01 || diff > 0.01 {
				active.SbWidth += diff * 0.3
				win.w.Invalidate()
			} else {
				active.SbWidth = targetWidth
			}

			width := unit.Dp(active.SbWidth)
			// Smoothly interpolate opacity based on current animated width
			alphaFactor := (active.SbWidth - 5.0) / 5.0
			if alphaFactor < 0 {
				alphaFactor = 0
			} else if alphaFactor > 1 {
				alphaFactor = 1
			}
			colorAlpha := uint8(60 + alphaFactor*140)

			sb := material.Scrollbar(win.theme, &active.scrollBar)
			sb.Track.Color.A = 0
			sb.Track.MajorPadding = 0
			sb.Track.MinorPadding = 0
			sb.Indicator.MinorWidth = width
			sb.Indicator.CornerRadius = unit.Dp(0) // No rounding!
			sb.Indicator.Color.A = colorAlpha
			sb.Indicator.HoverColor.A = colorAlpha

			sbWidthPx := gtx.Dp(width)

			targetX := baseTargetX
			if targetX+sbWidthPx > windowWidth {
				targetX = windowWidth - sbWidthPx
			}
			if targetX < 0 {
				targetX = 0
			}

			// Apply horizontal offset
			defer op.Offset(image.Pt(targetX, 0)).Push(gtx.Ops).Pop()

			return sb.Layout(gtx, layout.Vertical, vStart, vEnd)
		}),
	)
}

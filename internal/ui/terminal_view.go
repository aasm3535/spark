package ui

import (
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
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

	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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
			})
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

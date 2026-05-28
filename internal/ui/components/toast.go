package components

import (
	"image"
	"time"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type Toast struct {
	message   string
	startTime time.Time
	duration  time.Duration
	showUntil time.Time
}

func (t *Toast) Show(msg string, duration time.Duration) {
	t.message = msg
	t.startTime = time.Now()
	t.duration = duration
	t.showUntil = t.startTime.Add(duration)
}

func (t *Toast) Visible() bool {
	return time.Now().Before(t.showUntil)
}

func (t *Toast) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if !t.Visible() {
		return layout.Dimensions{}
	}

	fnt := font.Font{Typeface: th.Face, Weight: font.Normal}
	fontSize := unit.Sp(11)

	lbl := widget.Label{MaxLines: 1}

	// 1. Measure text dimensions first (ops stack is clean)
	measureMacro := op.Record(gtx.Ops)
	measureGtx := gtx
	measureGtx.Constraints.Min = image.Point{}
	textDims := lbl.Layout(measureGtx, th.Shaper, fnt, fontSize, t.message, op.CallOp{})
	measureMacro.Stop()

	// 2. Prepare text color call
	textColorMacro := op.Record(gtx.Ops)
	paint.ColorOp{Color: ColorText}.Add(gtx.Ops)
	textColorCall := textColorMacro.Stop()

	// 3. Calculate flat panel coordinates
	width := textDims.Size.X + 32
	height := textDims.Size.Y + 16

	if width < 140 {
		width = 140
	}

	x := (gtx.Constraints.Max.X - width) / 2
	y := gtx.Constraints.Max.Y - height - gtx.Dp(unit.Dp(30)) // Static position

	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	// 4. Push offset and draw
	off := op.Offset(image.Pt(x, y)).Push(gtx.Ops)

	rect := image.Rectangle{Max: image.Pt(width, height)}
	cr := clip.Rect(rect)

	// Draw solid flat background
	paint.FillShape(gtx.Ops, ColorTitleBar, cr.Op())

	// Draw text label centered
	textOff := op.Offset(image.Pt((width-textDims.Size.X)/2, (height-textDims.Size.Y)/2)).Push(gtx.Ops)
	lbl.Layout(gtx, th.Shaper, fnt, fontSize, t.message, textColorCall)
	textOff.Pop()

	// Draw flat 1dp border
	borderColor := blendColor(ColorTitleBar, 12)
	paint.FillShape(gtx.Ops, borderColor, clip.Stroke{
		Path:  cr.Path(),
		Width: float32(gtx.Dp(1)),
	}.Op())

	off.Pop()

	return layout.Dimensions{}
}

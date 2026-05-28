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
	showUntil time.Time
}

func (t *Toast) Show(msg string, duration time.Duration) {
	t.message = msg
	t.showUntil = time.Now().Add(duration)
}

func (t *Toast) Visible() bool {
	return time.Now().Before(t.showUntil)
}

func (t *Toast) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if !t.Visible() {
		return layout.Dimensions{}
	}

	fnt := font.Font{Typeface: th.Face, Weight: font.Normal}
	fontSize := unit.Sp(12)

	lbl := widget.Label{MaxLines: 1}

	macro := op.Record(gtx.Ops)
	measureGtx := gtx
	measureGtx.Constraints.Min = image.Point{}
	textDims := lbl.Layout(measureGtx, th.Shaper, fnt, fontSize, t.message, op.CallOp{})
	macro.Stop()

	width := textDims.Size.X + 40
	height := textDims.Size.Y + 20

	if width < 120 {
		width = 120
	}

	x := (gtx.Constraints.Max.X - width) / 2
	y := gtx.Constraints.Max.Y - height - 30

	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	off := op.Offset(image.Pt(x, y)).Push(gtx.Ops)

	rect := image.Rectangle{Max: image.Pt(width, height)}
	rr := clip.UniformRRect(rect, gtx.Dp(6))

	paint.FillShape(gtx.Ops, ColorTitleBar, rr.Op(gtx.Ops))

	textOff := op.Offset(image.Pt((width-textDims.Size.X)/2, (height-textDims.Size.Y)/2)).Push(gtx.Ops)
	macro2 := op.Record(gtx.Ops)
	paint.ColorOp{Color: ColorText}.Add(gtx.Ops)
	colorCall := macro2.Stop()
	lbl.Layout(gtx, th.Shaper, fnt, fontSize, t.message, colorCall)
	textOff.Pop()

	paint.FillShape(gtx.Ops, blendColor(ColorTitleBar, 30), clip.Stroke{
		Path:  rr.Path(gtx.Ops),
		Width: float32(gtx.Dp(1)),
	}.Op())

	off.Pop()

	return layout.Dimensions{}
}

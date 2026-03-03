package components

import (
	"image"
	"image/color"
	"math"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

const (
	TitleBarHeight = unit.Dp(32)
	btnW           = unit.Dp(46)
	btnH           = unit.Dp(32)
)

// TitleBar is the custom borderless title bar with minimize/maximize/close buttons.
type TitleBar struct {
	Close    widget.Clickable
	Minimize widget.Clickable
	Maximize widget.Clickable

	maximized bool
}

// Layout draws the title bar and handles window control clicks.
func (tb *TitleBar) Layout(gtx layout.Context, th *material.Theme, w *app.Window, title string) layout.Dimensions {
	height := gtx.Dp(TitleBarHeight)
	width := gtx.Constraints.Max.X

	// Background
	bgRect := image.Rectangle{Max: image.Pt(width, height)}
	paint.FillShape(gtx.Ops, ColorTitleBar, clip.Rect(bgRect).Op())

	// Drag region
	{
		st := clip.Rect(bgRect).Push(gtx.Ops)
		system.ActionInputOp(system.ActionMove).Add(gtx.Ops)
		st.Pop()
	}

	// Button click handling
	if tb.Close.Clicked(gtx) {
		w.Perform(system.ActionClose)
	}
	if tb.Minimize.Clicked(gtx) {
		w.Option(app.Minimized.Option())
	}
	if tb.Maximize.Clicked(gtx) {
		if tb.maximized {
			w.Option(app.Windowed.Option())
			tb.maximized = false
		} else {
			w.Option(app.Maximized.Option())
			tb.maximized = true
		}
	}

	bw := gtx.Dp(btnW)
	bh := gtx.Dp(btnH)
	totalBtns := bw * 3

	gtx.Constraints = layout.Exact(image.Pt(width, height))

	// Title label (left-aligned, vertically centred)
	{
		titleW := width - totalBtns
		off := op.Offset(image.Pt(0, 0)).Push(gtx.Ops)
		gtx2 := gtx
		gtx2.Constraints = layout.Exact(image.Pt(titleW, height))
		layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx2,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(16), Top: unit.Dp(7.5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, unit.Sp(12), title)
					lbl.Color = ColorTitleText
					lbl.Font = font.Font{
						Typeface: "Segoe UI, sans-serif",
						Weight:   font.Normal,
					}
					return lbl.Layout(gtx)
				})
			}),
		)
		off.Pop()
	}

	// Window control buttons
	btnX := width - totalBtns
	tb.drawMinimize(gtx, btnX, 0, bw, bh)
	tb.drawMaximize(gtx, btnX+bw, 0, bw, bh)
	tb.drawClose(gtx, btnX+bw*2, 0, bw, bh)

	return layout.Dimensions{Size: image.Pt(width, height)}
}

// ─── Button drawers ───────────────────────────────────────────────────────────

func (tb *TitleBar) drawMinimize(gtx layout.Context, x, y, bw, bh int) {
	r := image.Rectangle{Min: image.Pt(x, y), Max: image.Pt(x+bw, y+bh)}
	hovered := tb.Minimize.Hovered()

	if hovered {
		paint.FillShape(gtx.Ops, ColorBtnHoverNeutral, clip.Rect(r).Op())
	}

	offSt := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
	gtx2 := gtx
	gtx2.Constraints = layout.Exact(image.Pt(bw, bh))
	tb.Minimize.Layout(gtx2, func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Pt(bw, bh)}
	})
	offSt.Pop()

	col := symCol(hovered)
	cx := x + bw/2
	cy := y + bh/2
	drawFilledRect(gtx.Ops, cx-gtx.Dp(5), cy, gtx.Dp(10), gtx.Dp(1), col)
}

func (tb *TitleBar) drawMaximize(gtx layout.Context, x, y, bw, bh int) {
	r := image.Rectangle{Min: image.Pt(x, y), Max: image.Pt(x+bw, y+bh)}
	hovered := tb.Maximize.Hovered()

	if hovered {
		paint.FillShape(gtx.Ops, ColorBtnHoverNeutral, clip.Rect(r).Op())
	}

	offSt := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
	gtx2 := gtx
	gtx2.Constraints = layout.Exact(image.Pt(bw, bh))
	tb.Maximize.Layout(gtx2, func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Pt(bw, bh)}
	})
	offSt.Pop()

	col := symCol(hovered)
	cx := x + bw/2
	cy := y + bh/2
	sz := gtx.Dp(9)
	thick := gtx.Dp(1)

	if tb.maximized {
		off := gtx.Dp(3)
		drawHollowRect(gtx.Ops, cx-sz/2+off, cy-sz/2-off+1, sz-off, sz-off, thick, col)
		paint.FillShape(gtx.Ops, ColorTitleBar,
			clip.Rect{
				Min: image.Pt(cx-sz/2, cy-sz/2+1),
				Max: image.Pt(cx+sz/2-off, cy+sz/2+1),
			}.Op())
		drawHollowRect(gtx.Ops, cx-sz/2, cy-sz/2+1, sz-off, sz-off, thick, col)
	} else {
		drawHollowRect(gtx.Ops, cx-sz/2, cy-sz/2, sz, sz, thick, col)
	}
}

func (tb *TitleBar) drawClose(gtx layout.Context, x, y, bw, bh int) {
	r := image.Rectangle{Min: image.Pt(x, y), Max: image.Pt(x+bw, y+bh)}
	hovered := tb.Close.Hovered()

	if hovered {
		paint.FillShape(gtx.Ops, ColorBtnHoverClose, clip.Rect(r).Op())
	}

	offSt := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
	gtx2 := gtx
	gtx2.Constraints = layout.Exact(image.Pt(bw, bh))
	tb.Close.Layout(gtx2, func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Pt(bw, bh)}
	})
	offSt.Pop()

	col := symColClose(hovered)
	cx := float32(x + bw/2)
	cy := float32(y + bh/2)
	half := float32(gtx.Dp(5))
	thick := float32(gtx.Dp(1))

	drawLine(gtx.Ops, cx-half, cy-half, cx+half, cy+half, thick, col)
	drawLine(gtx.Ops, cx+half, cy-half, cx-half, cy+half, thick, col)
}

// ─── Drawing primitives ───────────────────────────────────────────────────────

func drawFilledRect(ops *op.Ops, x, y, w, h int, col color.NRGBA) {
	paint.FillShape(ops, col, clip.Rect{
		Min: image.Pt(x, y),
		Max: image.Pt(x+w, y+h),
	}.Op())
}

func drawHollowRect(ops *op.Ops, x, y, w, h, thick int, col color.NRGBA) {
	drawFilledRect(ops, x, y, w, thick, col)
	drawFilledRect(ops, x, y+h-thick, w, thick, col)
	drawFilledRect(ops, x, y, thick, h, col)
	drawFilledRect(ops, x+w-thick, y, thick, h, col)
}

func drawLine(ops *op.Ops, x1, y1, x2, y2, thick float32, col color.NRGBA) {
	dx := x2 - x1
	dy := y2 - y1
	length := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if length < 0.001 {
		return
	}

	ux := dx / length
	uy := dy / length
	px := -uy * (thick / 2)
	py := ux * (thick / 2)

	var path clip.Path
	path.Begin(ops)
	path.MoveTo(f32.Pt(x1+px, y1+py))
	path.LineTo(f32.Pt(x2+px, y2+py))
	path.LineTo(f32.Pt(x2-px, y2-py))
	path.LineTo(f32.Pt(x1-px, y1-py))
	path.Close()

	paint.FillShape(ops, col, clip.Outline{Path: path.End()}.Op())
}

// ─── Colour helpers ───────────────────────────────────────────────────────────

func symCol(hovered bool) color.NRGBA {
	if hovered {
		return color.NRGBA{R: 230, G: 230, B: 235, A: 255}
	}
	return color.NRGBA{R: 150, G: 150, B: 160, A: 255}
}

func symColClose(hovered bool) color.NRGBA {
	if hovered {
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	}
	return color.NRGBA{R: 150, G: 150, B: 160, A: 255}
}

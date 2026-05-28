package components

import (
	"image"
	"image/color"

	"gioui.org/app"
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
	TitleBarHeight = unit.Dp(40)
	btnW           = unit.Dp(46)
	btnH           = unit.Dp(40)
)

// TitleBarResult представляет результат взаимодействия с заголовком окна
type TitleBarResult struct {
	MenuClicked bool
}

// TitleBar is the custom borderless title bar with minimize/maximize/close buttons.
type TitleBar struct {
	Close    widget.Clickable
	Minimize widget.Clickable
	Maximize widget.Clickable
	Menu     widget.Clickable

	maximized bool
}

// Layout draws the title bar and handles window control clicks.
func (tb *TitleBar) Layout(gtx layout.Context, th *material.Theme, w *app.Window, title string, sidebarActive bool) (layout.Dimensions, TitleBarResult) {
	var res TitleBarResult
	height := gtx.Dp(TitleBarHeight)
	width := gtx.Constraints.Max.X

	// Background
	bgRect := image.Rectangle{Max: image.Pt(width, height)}
	paint.FillShape(gtx.Ops, ColorTitleBar, clip.Rect(bgRect).Op())

	// Тонкая нижняя граница тайтлбара
	borderColor := blendColor(ColorTitleBar, 8)
	drawFilledRect(gtx.Ops, 0, height-gtx.Dp(unit.Dp(1)), width, gtx.Dp(unit.Dp(1)), borderColor)

	// Drag region (excluding the menu button at x=0..40dp)
	{
		dragRect := image.Rect(gtx.Dp(unit.Dp(40)), 0, width, height)
		st := clip.Rect(dragRect).Push(gtx.Ops)
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
	if tb.Menu.Clicked(gtx) {
		res.MenuClicked = true
	}

	bw := gtx.Dp(btnW)
	bh := gtx.Dp(btnH)
	totalBtns := bw * 3

	gtx.Constraints = layout.Exact(image.Pt(width, height))

	// Отрисовка кнопки меню (гамбургер)
	tb.drawMenu(gtx, 0, 0, gtx.Dp(unit.Dp(40)), height)

	// Отрисовка заголовка (выравнивание по левому краю и центру Y)
	{
		// Измеряем размеры текста
		m := op.Record(gtx.Ops)
		lbl := material.Label(th, unit.Sp(12), title)
		lbl.Color = ColorTitleText
		lbl.Font = font.Font{
			Typeface: "Segoe UI, sans-serif",
			Weight:   font.Normal,
		}
		dims := lbl.Layout(gtx)
		call := m.Stop()

		// Вычисляем Y-координату для точного центра + ручная корректировка
		visualTuning := gtx.Dp(unit.Dp(12)) // Регулировка высоты текста (например: 0, 1, 2)
		yOffset := (height-dims.Size.Y)/2 + visualTuning

		// Отрисовываем текст со смещением 48dp от левого края (после кнопки меню)
		off := op.Offset(image.Pt(gtx.Dp(unit.Dp(48)), yOffset)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		off.Pop()
	}

	// Window control buttons
	btnX := width - totalBtns
	tb.drawMinimize(gtx, btnX, 0, bw, bh)
	tb.drawMaximize(gtx, btnX+bw, 0, bw, bh)
	tb.drawClose(gtx, btnX+bw*2, 0, bw, bh)

	return layout.Dimensions{Size: image.Pt(width, height)}, res
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
	scale := gtx.Dp(1)
	sz := 8 * scale
	thick := scale

	if tb.maximized {
		off := 2 * scale
		// Back square
		drawHollowRect(gtx.Ops, cx-sz/2+off, cy-sz/2, sz-off, sz-off, thick, col)
		// Clear front square area
		paint.FillShape(gtx.Ops, ColorTitleBar, clip.Rect{
			Min: image.Pt(cx-sz/2, cy-sz/2+off),
			Max: image.Pt(cx-sz/2+sz-off, cy-sz/2+sz),
		}.Op())
		// Front square
		drawHollowRect(gtx.Ops, cx-sz/2, cy-sz/2+off, sz-off, sz-off, thick, col)
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
	cx := x + bw/2
	cy := y + bh/2
	scale := gtx.Dp(1)
	half := 4 * scale

	drawSharpCross(gtx.Ops, cx, cy, half, scale, col)
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

func drawSharpCross(ops *op.Ops, cx, cy, half, scale int, col color.NRGBA) {
	for i := -half; i < half; i += scale {
		// Diagonal 1: top-left to bottom-right
		drawFilledRect(ops, cx+i, cy+i, scale, scale, col)
		// Diagonal 2: top-right to bottom-left
		drawFilledRect(ops, cx-i-scale, cy+i, scale, scale, col)
	}
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

// drawMenu рисует кнопку гамбургер-меню для скрытия/показа боковой панели
func (tb *TitleBar) drawMenu(gtx layout.Context, x, y, bw, bh int) {
	r := image.Rectangle{Min: image.Pt(x, y), Max: image.Pt(x+bw, y+bh)}
	hovered := tb.Menu.Hovered()

	if hovered {
		paint.FillShape(gtx.Ops, ColorBtnHoverNeutral, clip.Rect(r).Op())
	}

	offSt := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
	gtx2 := gtx
	gtx2.Constraints = layout.Exact(image.Pt(bw, bh))
	tb.Menu.Layout(gtx2, func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Pt(bw, bh)}
	})
	offSt.Pop()

	col := symCol(hovered)
	cx := x + bw/2
	cy := y + bh/2
	scale := gtx.Dp(1)

	// Три горизонтальные полоски гамбургера
	lineW := 14 * scale
	lineH := scale
	drawFilledRect(gtx.Ops, cx-lineW/2, cy-4*scale, lineW, lineH, col)
	drawFilledRect(gtx.Ops, cx-lineW/2, cy, lineW, lineH, col)
	drawFilledRect(gtx.Ops, cx-lineW/2, cy+4*scale, lineW, lineH, col)
}

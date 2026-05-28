package components

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// Sidebar представляет боковую панель для быстрого доступа к функциям
type Sidebar struct {
	NewTab widget.Clickable
	CmdPal widget.Clickable
}

// SidebarResult содержит события кликов по кнопкам панели
type SidebarResult struct {
	NewTabClicked bool
	CmdPalClicked bool
}

// Layout рисует боковую панель и обрабатывает нажатия
func (s *Sidebar) Layout(gtx layout.Context, th *material.Theme) (layout.Dimensions, SidebarResult) {
	var res SidebarResult

	if s.NewTab.Clicked(gtx) {
		res.NewTabClicked = true
	}
	if s.CmdPal.Clicked(gtx) {
		res.CmdPalClicked = true
	}

	width := gtx.Dp(unit.Dp(48))
	height := gtx.Constraints.Max.Y

	// Фоновый цвет панели (чуть темнее тайтлбара)
	bgCol := blendColor(ColorTitleBar, -2)
	paint.FillShape(gtx.Ops, bgCol, clip.Rect{Max: image.Pt(width, height)}.Op())

	// Правая граница панели
	borderColor := blendColor(ColorTitleBar, 10)
	drawFilledRect(gtx.Ops, width-gtx.Dp(unit.Dp(1)), 0, gtx.Dp(unit.Dp(1)), height, borderColor)

	// Разметка кнопок по вертикали
	layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Middle,
	}.Layout(gtx,
		// Отступ сверху
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(width, gtx.Dp(unit.Dp(8)))}
		}),
		// Кнопка новой вкладки
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Left:  unit.Dp(6),
				Right: unit.Dp(6),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return s.layoutButton(gtx, th, &s.NewTab, "+")
			})
		}),
		// Промежуток
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(width, gtx.Dp(unit.Dp(8)))}
		}),
		// Кнопка командной палитры
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Left:  unit.Dp(6),
				Right: unit.Dp(6),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return s.layoutButton(gtx, th, &s.CmdPal, ">")
			})
		}),
	)

	return layout.Dimensions{Size: image.Pt(width, height)}, res
}

// layoutButton рисует отдельную кнопку с hover-эффектом
func (s *Sidebar) layoutButton(gtx layout.Context, th *material.Theme, click *widget.Clickable, symbol string) layout.Dimensions {
	size := gtx.Dp(unit.Dp(36))
	gtx.Constraints = layout.Exact(image.Pt(size, size))

	hovered := click.Hovered()
	bgCol := color.NRGBA{R: 0, G: 0, B: 0, A: 0}
	if hovered {
		bgCol = ColorBtnHoverNeutral
	}

	rr := clip.RRect{
		Rect: image.Rectangle{Max: image.Pt(size, size)},
		NE:   gtx.Dp(unit.Dp(6)), NW: gtx.Dp(unit.Dp(6)),
		SE:   gtx.Dp(unit.Dp(6)), SW: gtx.Dp(unit.Dp(6)),
	}

	m := op.Record(gtx.Ops)
	dims := click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, unit.Sp(16), symbol)
			lbl.Color = ColorTitleText
			if hovered {
				lbl.Color = ColorText
			}
			return lbl.Layout(gtx)
		})
	})
	call := m.Stop()

	paint.FillShape(gtx.Ops, bgCol, rr.Op(gtx.Ops))
	call.Add(gtx.Ops)

	return dims
}

package components

import (
	"fmt"
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// Sidebar представляет вертикальную панель вкладок и кнопок управления
type Sidebar struct {
	NewTab widget.Clickable
	CmdPal widget.Clickable
	Tabs   []*TabState
}

// SidebarResult возвращает события взаимодействия с боковой панелью
type SidebarResult struct {
	NewTabClicked bool
	CmdPalClicked bool
	TabSwitchedTo int
	TabClosedIdx  int
}

// Layout рисует боковую панель с вертикальным списком вкладок
func (s *Sidebar) Layout(
	gtx layout.Context,
	th *material.Theme,
	activeIdx int,
	titles []string,
) (layout.Dimensions, SidebarResult) {
	res := SidebarResult{
		TabSwitchedTo: -1,
		TabClosedIdx:  -1,
	}

	if s.NewTab.Clicked(gtx) {
		res.NewTabClicked = true
	}
	if s.CmdPal.Clicked(gtx) {
		res.CmdPalClicked = true
	}

	width := gtx.Dp(unit.Dp(180)) // Ширина боковой панели
	height := gtx.Constraints.Max.Y

	// Фоновый цвет панели (чуть темнее тайтлбара)
	bgCol := blendColor(ColorTitleBar, -2)
	paint.FillShape(gtx.Ops, bgCol, clip.Rect{Max: image.Pt(width, height)}.Op())

	// Тонкая правая граница
	borderColor := blendColor(ColorTitleBar, 10)
	drawFilledRect(gtx.Ops, width-gtx.Dp(unit.Dp(1)), 0, gtx.Dp(unit.Dp(1)), height, borderColor)

	// Ограничиваем область рисования
	defer clip.Rect{Max: image.Pt(width, height)}.Push(gtx.Ops).Pop()

	// Разметка элементов
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Заголовок секции TABS и кнопка "+"
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Top:    unit.Dp(10),
				Bottom: unit.Dp(6),
				Left:   unit.Dp(12),
				Right:  unit.Dp(8),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{
					Axis:      layout.Horizontal,
					Alignment: layout.Middle,
					Spacing:   layout.SpaceBetween,
				}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, unit.Sp(10), "ВКЛАДКИ")
						lbl.Color = blendColor(ColorTitleText, -20)
						lbl.Font.Weight = font.Bold
						return lbl.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return s.layoutHeaderButton(gtx, th, &s.NewTab, "+")
					}),
				)
			})
		}),

		// Вертикальный список вкладок
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			var tabChildren []layout.FlexChild

			for i, tab := range s.Tabs {
				i := i
				tab := tab
				isActive := i == activeIdx

				title := fmt.Sprintf("Вкладка %d", i+1)
				if i < len(titles) && titles[i] != "" {
					title = titles[i]
				}

				tabChildren = append(tabChildren, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{
						Left:  unit.Dp(8),
						Right: unit.Dp(8),
						Top:   unit.Dp(2),
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						dims, clicked, closed := s.layoutTabItem(gtx, th, tab, title, isActive)
						if clicked {
							res.TabSwitchedTo = i
						}
						if closed {
							res.TabClosedIdx = i
						}
						return dims
					})
				}))
			}

			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, tabChildren...)
		}),

		// Нижняя кнопка вызова командной палитры
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Bottom: unit.Dp(10),
				Left:   unit.Dp(8),
				Right:  unit.Dp(8),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return s.layoutFooterButton(gtx, th, &s.CmdPal, "> Команды")
			})
		}),
	)

	return layout.Dimensions{Size: image.Pt(width, height)}, res
}

// layoutTabItem рисует одну вертикальную вкладку
func (s *Sidebar) layoutTabItem(
	gtx layout.Context,
	th *material.Theme,
	tab *TabState,
	title string,
	isActive bool,
) (layout.Dimensions, bool, bool) {
	var clicked, closed bool
	if tab.BtnClick.Clicked(gtx) {
		clicked = true
	}
	if tab.BtnClose.Clicked(gtx) {
		closed = true
	}

	width := gtx.Constraints.Max.X
	height := gtx.Dp(unit.Dp(32))
	gtx.Constraints = layout.Exact(image.Pt(width, height))

	// Цвета в зависимости от активности
	bgCol := color.NRGBA{R: 0, G: 0, B: 0, A: 0}
	if isActive {
		bgCol = ColorTabActiveBg
	} else if tab.BtnClick.Hovered() {
		bgCol = ColorTabHoverBg
	}

	m := op.Record(gtx.Ops)
	dims := tab.BtnClick.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			// Индикатор активности слева
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				w := gtx.Dp(unit.Dp(3))
				h := gtx.Dp(unit.Dp(16))
				if isActive {
					drawFilledRect(gtx.Ops, 0, (height-h)/2, w, h, ColorText)
				}
				return layout.Dimensions{Size: image.Pt(w+gtx.Dp(unit.Dp(6)), height)}
			}),
			// Название вкладки
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, unit.Sp(11.5), title)
				lbl.Font.Typeface = "Segoe UI, sans-serif"
				if isActive {
					lbl.Color = ColorText
					lbl.Font.Weight = font.Medium
				} else {
					lbl.Color = ColorTitleText
				}
				lbl.MaxLines = 1
				return lbl.Layout(gtx)
			}),
			// Кнопка закрытия (показывается при наведении или активности)
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if tab.BtnClick.Hovered() || isActive {
					return layoutCloseButton(gtx, tab, isActive)
				}
				return layout.Dimensions{}
			}),
		)
	})
	call := m.Stop()

	// Рисуем скругленный фон вкладки
	rr := clip.RRect{
		Rect: image.Rectangle{Max: image.Pt(width, height)},
		NE:   gtx.Dp(unit.Dp(4)), NW: gtx.Dp(unit.Dp(4)),
		SE:   gtx.Dp(unit.Dp(4)), SW: gtx.Dp(unit.Dp(4)),
	}
	paint.FillShape(gtx.Ops, bgCol, rr.Op(gtx.Ops))
	call.Add(gtx.Ops)

	return dims, clicked, closed
}

// layoutHeaderButton рисует компактную кнопку "+" в заголовке
func (s *Sidebar) layoutHeaderButton(gtx layout.Context, th *material.Theme, click *widget.Clickable, text string) layout.Dimensions {
	size := gtx.Dp(unit.Dp(24))
	gtx.Constraints = layout.Exact(image.Pt(size, size))

	hovered := click.Hovered()
	bgCol := color.NRGBA{R: 0, G: 0, B: 0, A: 0}
	if hovered {
		bgCol = ColorBtnHoverNeutral
	}

	rr := clip.RRect{
		Rect: image.Rectangle{Max: image.Pt(size, size)},
		NE:   gtx.Dp(unit.Dp(4)), NW: gtx.Dp(unit.Dp(4)),
		SE:   gtx.Dp(unit.Dp(4)), SW: gtx.Dp(unit.Dp(4)),
	}

	m := op.Record(gtx.Ops)
	dims := click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, unit.Sp(14), text)
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

// layoutFooterButton рисует нижнюю текстовую кнопку "> Команды"
func (s *Sidebar) layoutFooterButton(gtx layout.Context, th *material.Theme, click *widget.Clickable, text string) layout.Dimensions {
	width := gtx.Constraints.Max.X
	height := gtx.Dp(unit.Dp(32))
	gtx.Constraints = layout.Exact(image.Pt(width, height))

	hovered := click.Hovered()
	bgCol := color.NRGBA{R: 0, G: 0, B: 0, A: 0}
	if hovered {
		bgCol = ColorBtnHoverNeutral
	}

	rr := clip.RRect{
		Rect: image.Rectangle{Max: image.Pt(width, height)},
		NE:   gtx.Dp(unit.Dp(4)), NW: gtx.Dp(unit.Dp(4)),
		SE:   gtx.Dp(unit.Dp(4)), SW: gtx.Dp(unit.Dp(4)),
	}

	m := op.Record(gtx.Ops)
	dims := click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Left: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, unit.Sp(11.5), text)
					lbl.Font.Typeface = "Segoe UI, sans-serif"
					lbl.Color = ColorTitleText
					if hovered {
						lbl.Color = ColorText
					}
					return lbl.Layout(gtx)
				}),
			)
		})
	})
	call := m.Stop()

	paint.FillShape(gtx.Ops, bgCol, rr.Op(gtx.Ops))
	call.Add(gtx.Ops)

	return dims
}

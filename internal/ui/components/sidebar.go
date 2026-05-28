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

// Sidebar представляет вертикальную панель вкладок без скруглений (плоский стиль)
type Sidebar struct {
	NewTab widget.Clickable
	CmdPal widget.Clickable
	Tabs   []*TabState
}

type SidebarResult struct {
	NewTabClicked bool
	CmdPalClicked bool
	TabSwitchedTo int
	TabClosedIdx  int
}

// Layout рисует плоскую боковую панель
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

	width := gtx.Dp(unit.Dp(160)) // Ширина сайдбара
	height := gtx.Constraints.Max.Y

	// Фон сайдбара (темнее основного фона)
	bgCol := blendColor(ColorTitleBar, -2)
	paint.FillShape(gtx.Ops, bgCol, clip.Rect{Max: image.Pt(width, height)}.Op())

	// Правая граница сайдбара
	borderColor := blendColor(ColorTitleBar, 8)
	drawFilledRect(gtx.Ops, width-gtx.Dp(unit.Dp(1)), 0, gtx.Dp(unit.Dp(1)), height, borderColor)

	// Ограничиваем область вывода
	defer clip.Rect{Max: image.Pt(width, height)}.Push(gtx.Ops).Pop()

	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Заголовок панели с кнопкой "+"
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Top:    unit.Dp(8),
				Bottom: unit.Dp(8),
				Left:   unit.Dp(12),
				Right:  unit.Dp(8),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{
					Axis:      layout.Horizontal,
					Alignment: layout.Middle,
					Spacing:   layout.SpaceBetween,
				}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, unit.Sp(10), "TABS")
						lbl.Color = blendColor(ColorTitleText, -30)
						lbl.Font.Weight = font.Bold
						return lbl.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return s.layoutHeaderButton(gtx, th, &s.NewTab, "+")
					}),
				)
			})
		}),

		// Список вкладок (плоские элементы во всю ширину)
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			var tabChildren []layout.FlexChild

			for i, tab := range s.Tabs {
				i := i
				tab := tab
				isActive := i == activeIdx

				title := fmt.Sprintf("Tab %d", i+1)
				if i < len(titles) && titles[i] != "" {
					title = titles[i]
				}

				tabChildren = append(tabChildren, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{
						Top: unit.Dp(1), // Минимальный разделитель
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

		// Нижняя панель с кнопкой командной палитры
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Bottom: unit.Dp(8),
				Left:   unit.Dp(8),
				Right:  unit.Dp(8),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return s.layoutFooterButton(gtx, th, &s.CmdPal, "> commands")
			})
		}),
	)

	return layout.Dimensions{Size: image.Pt(width, height)}, res
}

// layoutTabItem рисует плоскую вкладку (стиль без скруглений)
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

	// Фон вкладки
	bgCol := color.NRGBA{R: 0, G: 0, B: 0, A: 0}
	if isActive {
		bgCol = ColorTabActiveBg
	} else if tab.BtnClick.Hovered() {
		bgCol = ColorTabHoverBg
	}

	m := op.Record(gtx.Ops)
	dims := tab.BtnClick.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			// Индикатор активности слева (тонкая линия 3dp)
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				w := gtx.Dp(unit.Dp(3))
				h := gtx.Dp(unit.Dp(16))
				if isActive {
					drawFilledRect(gtx.Ops, 0, (height-h)/2, w, h, ColorText)
				}
				return layout.Dimensions{Size: image.Pt(w+gtx.Dp(unit.Dp(8)), height)}
			}),
			// Текст вкладки
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, unit.Sp(11), title)
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
			// Кнопка закрытия
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if tab.BtnClick.Hovered() || isActive {
					return layoutCloseButton(gtx, tab, isActive)
				}
				return layout.Dimensions{}
			}),
		)
	})
	call := m.Stop()

	// Заливаем плоский фон без скруглений
	paint.FillShape(gtx.Ops, bgCol, clip.Rect{Max: image.Pt(width, height)}.Op())
	call.Add(gtx.Ops)

	return dims, clicked, closed
}

// layoutHeaderButton рисует плоскую кнопку "+"
func (s *Sidebar) layoutHeaderButton(gtx layout.Context, th *material.Theme, click *widget.Clickable, text string) layout.Dimensions {
	size := gtx.Dp(unit.Dp(24))
	gtx.Constraints = layout.Exact(image.Pt(size, size))

	hovered := click.Hovered()
	bgCol := color.NRGBA{R: 0, G: 0, B: 0, A: 0}
	if hovered {
		bgCol = ColorBtnHoverNeutral
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

	// Заливка фона без скруглений
	paint.FillShape(gtx.Ops, bgCol, clip.Rect{Max: image.Pt(size, size)}.Op())
	call.Add(gtx.Ops)

	return dims
}

// layoutFooterButton рисует нижнюю кнопку без скруглений
func (s *Sidebar) layoutFooterButton(gtx layout.Context, th *material.Theme, click *widget.Clickable, text string) layout.Dimensions {
	width := gtx.Constraints.Max.X
	height := gtx.Dp(unit.Dp(32))
	gtx.Constraints = layout.Exact(image.Pt(width, height))

	hovered := click.Hovered()
	bgCol := color.NRGBA{R: 0, G: 0, B: 0, A: 0}
	if hovered {
		bgCol = ColorBtnHoverNeutral
	}

	m := op.Record(gtx.Ops)
	dims := click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Left: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, unit.Sp(11), text)
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

	// Заливка фона без скруглений
	paint.FillShape(gtx.Ops, bgCol, clip.Rect{Max: image.Pt(width, height)}.Op())
	call.Add(gtx.Ops)

	return dims
}

package components

import (
	"fmt"
	"image"
	"image/color"

	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// Sidebar представляет вертикальную панель вкладок без скруглений и лишних элементов
type Sidebar struct {
	NewTab widget.Clickable
	CmdPal widget.Clickable
	Tabs   []*TabState

	// Dragging state
	DragTag   int
	dragging  bool
	dragStart float32
}

type SidebarResult struct {
	NewTabClicked bool
	CmdPalClicked bool
	TabSwitchedTo int
	TabClosedIdx  int
	WidthDeltaDp  unit.Dp
}

// Layout рисует плоскую боковую панель
func (s *Sidebar) Layout(
	gtx layout.Context,
	th *material.Theme,
	widthDp unit.Dp,
	activeIdx int,
	titles []string,
	descriptions []string,
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

	width := gtx.Dp(widthDp)
	height := gtx.Constraints.Max.Y

	// Process pointer events on the drag area (6dp wide interaction area on the right edge)
	{
		dragAreaW := gtx.Dp(unit.Dp(6))
		dragArea := image.Rect(width-dragAreaW/2, 0, width+dragAreaW/2, height)
		stack := clip.Rect(dragArea).Push(gtx.Ops)
		event.Op(gtx.Ops, &s.DragTag)
		pointer.CursorColResize.Add(gtx.Ops)
		stack.Pop()
	}

	dragFilter := pointer.Filter{
		Target: &s.DragTag,
		Kinds:  pointer.Press | pointer.Release | pointer.Drag | pointer.Move,
	}

	for {
		ev, ok := gtx.Event(dragFilter)
		if !ok {
			break
		}
		if x, ok := ev.(pointer.Event); ok {
			switch x.Kind {
			case pointer.Press:
				s.dragging = true
				s.dragStart = x.Position.X
			case pointer.Drag:
				if s.dragging {
					deltaPx := x.Position.X - s.dragStart
					deltaDp := gtx.Metric.PxToDp(int(deltaPx))
					if deltaDp != 0 {
						res.WidthDeltaDp = deltaDp
						s.dragStart += float32(gtx.Dp(deltaDp))
					}
				}
			case pointer.Release, pointer.Cancel:
				s.dragging = false
			}
		}
	}

	// Фон сайдбара (совпадает с тайтлбаром)
	bgCol := ColorTitleBar
	paint.FillShape(gtx.Ops, bgCol, clip.Rect{Max: image.Pt(width, height)}.Op())

	// Ограничиваем область вывода
	defer clip.Rect{Max: image.Pt(width, height)}.Push(gtx.Ops).Pop()

	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Верхний отступ вместо заголовков
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(width, gtx.Dp(unit.Dp(0)))}
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

				desc := ""
				if i < len(descriptions) {
					desc = descriptions[i]
				}
				tabChildren = append(tabChildren, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					dims, clicked, closed := s.layoutTabItem(gtx, th, tab, title, desc, isActive)
					if clicked {
						res.TabSwitchedTo = i
					}
					if closed {
						res.TabClosedIdx = i
					}
					return dims
				}))
			}

			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, tabChildren...)
		}),

		// Кнопка "+" в самом низу для добавления новых вкладок
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Bottom: unit.Dp(8),
				Left:   unit.Dp(8),
				Right:  unit.Dp(8),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return s.layoutFooterButton(gtx, th, &s.NewTab, "+ New Session")
			})
		}),
	)

	// Правая граница сайдбара (рисуется поверх вкладок)
	borderColor := blendColor(ColorTitleBar, 8)
	drawFilledRect(gtx.Ops, width-gtx.Dp(unit.Dp(1)), 0, gtx.Dp(unit.Dp(1)), height, borderColor)

	return layout.Dimensions{Size: image.Pt(width, height)}, res
}

// layoutTabItem рисует плоскую вкладку с заголовком и описанием (без скруглений)
func (s *Sidebar) layoutTabItem(
	gtx layout.Context,
	th *material.Theme,
	tab *TabState,
	title string,
	desc string,
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
	height := gtx.Dp(unit.Dp(48)) // Высота под 2 строки
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
			// Левый отступ
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Dimensions{Size: image.Pt(gtx.Dp(unit.Dp(12)), height)}
			}),
			// Текст в два ряда (название + ласт контент)
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				// РЕГУЛИРОВКА ВЕРТИКАЛЬНОГО СМЕЩЕНИЯ ТЕКСТА
				// Изменяйте это значение (например: -2, 0, 2), чтобы сдвинуть текст по Y
				visualTuningY := unit.Dp(8)

				return layout.Inset{Top: visualTuningY}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical, Alignment: layout.Start}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(th, unit.Sp(12), title)
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
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if desc == "" {
								desc = "no activity"
							}
							lbl := material.Label(th, unit.Sp(9.5), desc)
							lbl.Font.Typeface = "Segoe UI, sans-serif"
							lbl.Color = blendColor(ColorTitleText, -30)
							lbl.MaxLines = 1
							return lbl.Layout(gtx)
						}),
					)
				})
			}),
			// Кнопка закрытия (резервирует место, иконка рисуется при наведении)
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layoutCloseButton(gtx, tab, isActive)
			}),
		)
	})
	call := m.Stop()

	// Рисуем плоский фон
	paint.FillShape(gtx.Ops, bgCol, clip.Rect{Max: image.Pt(width, height)}.Op())
	call.Add(gtx.Ops)

	// Горизонтальный разделитель снизу вкладки
	sepColor := blendColor(ColorTitleBar, 8)
	drawFilledRect(gtx.Ops, 0, height-gtx.Dp(unit.Dp(1)), width, gtx.Dp(unit.Dp(1)), sepColor)

	return dims, clicked, closed
}

// layoutFooterButton рисует плоскую кнопку "+"/"New Session"
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
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, unit.Sp(11), text)
			lbl.Font.Typeface = "Segoe UI, sans-serif"
			lbl.Color = ColorTitleText
			if hovered {
				lbl.Color = ColorText
			}
			return lbl.Layout(gtx)
		})
	})
	call := m.Stop()

	paint.FillShape(gtx.Ops, bgCol, clip.Rect{Max: image.Pt(width, height)}.Op())
	call.Add(gtx.Ops)

	return dims
}

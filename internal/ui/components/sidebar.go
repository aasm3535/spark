package components

import (
	"fmt"
	"image"
	"image/color"
	"strings"

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
	DragTag        int
	dragging       bool
	dragStart      float32
	dragStartWidth unit.Dp
}

type SidebarResult struct {
	NewTabClicked bool
	CmdPalClicked bool
	TabSwitchedTo int
	TabClosedIdx  int
	WidthDeltaDp  unit.Dp
	Dragging      bool
}

// Layout рисует плоскую боковую панель
func (s *Sidebar) Layout(
	gtx layout.Context,
	th *material.Theme,
	widthDp unit.Dp,
	activeIdx int,
	titles []string,
	descriptions []string,
	sttRecording bool,
	sttTranscribing bool,
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

	height := gtx.Constraints.Max.Y

	// Process pointer events on the drag area (6dp wide interaction area on the right edge)
	{
		width := gtx.Dp(widthDp)
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

	widthDpVar := widthDp

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
				s.dragStartWidth = widthDp
			case pointer.Drag:
				if s.dragging {
					deltaPx := x.Position.X - s.dragStart
					deltaDp := gtx.Metric.PxToDp(int(deltaPx))
					targetWidth := s.dragStartWidth + deltaDp
					if targetWidth < 120 {
						targetWidth = 120
					}
					if targetWidth > 400 {
						targetWidth = 400
					}
					res.WidthDeltaDp = targetWidth - widthDp
					widthDpVar = targetWidth
				}
			case pointer.Release, pointer.Cancel:
				s.dragging = false
			}
		}
	}

	res.Dragging = s.dragging
	width := gtx.Dp(widthDpVar)

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
					isRecordingTab := isActive && sttRecording
					isTranscribingTab := isActive && sttTranscribing
					dims, clicked, closed := s.layoutTabItem(
						gtx, th, tab, title, desc, isActive, i == 0,
						isRecordingTab, isTranscribingTab,
					)
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
	)

	// Правая граница сайдбара (рисуется поверх вкладок, кроме активной)
	borderColor := blendColor(ColorTitleBar, 8)
	tabHeight := gtx.Dp(unit.Dp(48))
	if activeIdx >= 0 && activeIdx < len(s.Tabs) {
		activeTabTop := activeIdx * tabHeight
		activeTabBottom := (activeIdx + 1) * tabHeight
		if activeTabTop > 0 {
			drawFilledRect(gtx.Ops, width-gtx.Dp(unit.Dp(1)), 0, gtx.Dp(unit.Dp(1)), activeTabTop, borderColor)
		}
		if activeTabBottom < height {
			drawFilledRect(gtx.Ops, width-gtx.Dp(unit.Dp(1)), activeTabBottom, gtx.Dp(unit.Dp(1)), height-activeTabBottom, borderColor)
		}
	} else {
		drawFilledRect(gtx.Ops, width-gtx.Dp(unit.Dp(1)), 0, gtx.Dp(unit.Dp(1)), height, borderColor)
	}

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
	isFirst bool,
	isRecording bool,
	isTranscribing bool,
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
							lbl.Font.Typeface = "Segoe UI, Segoe UI Emoji, Apple Color Emoji, Noto Color Emoji, DejaVu Sans, Segoe UI Symbol, sans-serif"
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
							lbl.Font.Typeface = "Segoe UI, Segoe UI Emoji, Apple Color Emoji, Noto Color Emoji, DejaVu Sans, Segoe UI Symbol, sans-serif"
							isAgent := false
							for _, r := range []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'} {
								if strings.HasPrefix(desc, string(r)) {
									isAgent = true
									break
								}
							}
							if strings.Contains(desc, "Claude Code") ||
								strings.Contains(desc, "Opencode") ||
								strings.Contains(desc, "Pi") {
								isAgent = true
							}
							if isAgent {
								lbl.Color = ColorText
								lbl.Font.Weight = font.Medium
							} else {
								lbl.Color = blendColor(ColorTitleText, -30)
							}
							lbl.MaxLines = 1
							return lbl.Layout(gtx)
						}),
					)
				})
			}),
			// Кнопка закрытия
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layoutCloseButton(gtx, tab, isActive)
			}),
		)
	})
	call := m.Stop()

	// Рисуем плоский фон
	paint.FillShape(gtx.Ops, bgCol, clip.Rect{Max: image.Pt(width, height)}.Op())
	call.Add(gtx.Ops)

	// Горизонтальные разделители для выделения активной вкладки
	if isActive {
		sepColor := blendColor(ColorTitleBar, 8)
		if !isFirst {
			drawFilledRect(gtx.Ops, 0, 0, width, gtx.Dp(unit.Dp(1)), sepColor)
		}
		drawFilledRect(gtx.Ops, 0, height-gtx.Dp(unit.Dp(1)), width, gtx.Dp(unit.Dp(1)), sepColor)
	}

	return dims, clicked, closed
}

// layoutFooterButton рисует кнопку "+ New Session" со скруглением и бордером
func (s *Sidebar) layoutFooterButton(gtx layout.Context, th *material.Theme, click *widget.Clickable, text string) layout.Dimensions {
	hovered := click.Hovered()

	// Measure text to get button size
	macro := op.Record(gtx.Ops)
	measureGtx := gtx
	measureGtx.Constraints.Min = image.Point{}
	lbl := material.Label(th, unit.Sp(11), text)
	textDims := lbl.Layout(measureGtx)
	macro.Stop()

	hPad := gtx.Dp(unit.Dp(14))
	vPad := gtx.Dp(unit.Dp(7))
	btnW := textDims.Size.X + hPad*2
	btnH := textDims.Size.Y + vPad*2

	// Center the button within available space
	availW := gtx.Constraints.Max.X
	availH := gtx.Dp(unit.Dp(32))
	offsetX := (availW - btnW) / 2
	offsetY := (availH - btnH) / 2

	// Record all visual ops (bg + border + text) so we can place them after
	m := op.Record(gtx.Ops)

	// Background
	radius := gtx.Dp(unit.Dp(4))
	rr := clip.RRect{Rect: image.Rect(0, 0, btnW, btnH), NE: radius, NW: radius, SE: radius, SW: radius}

	bgCol := blendColor(ColorTitleBar, 10)
	if hovered {
		bgCol = blendColor(ColorTitleBar, 22)
	}
	paint.FillShape(gtx.Ops, bgCol, rr.Op(gtx.Ops))

	// Border
	borderCol := blendColor(ColorTitleBar, 18)
	paint.FillShape(gtx.Ops, borderCol, clip.Stroke{
		Path:  rr.Path(gtx.Ops),
		Width: float32(gtx.Dp(unit.Dp(1))),
	}.Op())

	// Text centered inside the button
	textX := (btnW - textDims.Size.X) / 2
	textY := (btnH - textDims.Size.Y) / 2
	textOff := op.Offset(image.Pt(textX, textY)).Push(gtx.Ops)
	txtLbl := material.Label(th, unit.Sp(11), text)
	txtLbl.Color = blendColor(ColorTitleText, 40)
	if hovered {
		txtLbl.Color = ColorText
	}
	txtLbl.Layout(gtx)
	textOff.Pop()

	call := m.Stop()

	// Clickable covers the button area
	gtx2 := gtx
	gtx2.Constraints = layout.Exact(image.Pt(btnW, btnH))
	_ = click.Layout(gtx2, func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Pt(btnW, btnH)}
	})

	// Offset everything to center and replay visuals
	off := op.Offset(image.Pt(offsetX, offsetY)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	off.Pop()

	return layout.Dimensions{Size: image.Pt(availW, availH)}
}



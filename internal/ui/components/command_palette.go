package components

import (
	"image"
	"image/color"
	"strings"

	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"yutug.lol/spark/internal/config"
)

type CommandItem struct {
	Name     string
	Shortcut string
	Action   config.Action
}

// CommandPalette provides a searchable list of commands.
type CommandPalette struct {
	Editor  widget.Editor
	focused bool

	List        widget.List
	Items       []CommandItem
	Filtered    []CommandItem
	ActiveIndex int
	lastText    string
}

// CommandPaletteResult reports events from the command palette.
type CommandPaletteResult struct {
	Action    config.Action
	Closed    bool
	Submitted bool
}

// InitDefaults sets up the static list of commands.
func (c *CommandPalette) InitDefaults() {
	if len(c.Items) > 0 {
		return
	}
	c.Items = []CommandItem{
		{"New Tab", "Ctrl+Shift+T", config.ActionNewTab},
		{"Close Tab", "Ctrl+Shift+W", config.ActionCloseTab},
		{"Next Tab", "Ctrl+PageDown", config.ActionNextTab},
		{"Previous Tab", "Ctrl+PageUp", config.ActionPrevTab},
		{"Scroll Up", "Shift+Up", config.ActionScrollUp},
		{"Scroll Down", "Shift+Down", config.ActionScrollDown},
		{"Scroll Page Up", "Shift+PageUp", config.ActionScrollPageUp},
		{"Scroll Page Down", "Shift+PageDown", config.ActionScrollPageDown},
	}
	c.List.Axis = layout.Vertical
	c.Filtered = c.Items
}

// MoveSelection updates the currently selected item and scrolls to it.
func (c *CommandPalette) MoveSelection(delta int) {
	if len(c.Filtered) == 0 {
		return
	}
	c.ActiveIndex += delta
	if c.ActiveIndex < 0 {
		c.ActiveIndex = len(c.Filtered) - 1
	} else if c.ActiveIndex >= len(c.Filtered) {
		c.ActiveIndex = 0
	}
	c.List.ScrollTo(c.ActiveIndex)
}

// Layout draws the command palette overlay if active.
func (c *CommandPalette) Layout(
	gtx layout.Context,
	th *material.Theme,
	active bool,
) (layout.Dimensions, CommandPaletteResult) {
	var result CommandPaletteResult

	if !active {
		c.focused = false
		return layout.Dimensions{}, result
	}

	c.InitDefaults()
	c.Editor.Submit = true
	c.Editor.SingleLine = true

	if !c.focused {
		gtx.Execute(key.FocusCmd{Tag: &c.Editor})
		c.focused = true
		c.Editor.SetText("")
		c.lastText = ""
		c.Filtered = c.Items
		c.ActiveIndex = 0
	}

	// Intercept keys on the editor before the editor processes them
	for {
		ev, ok := gtx.Event(
			key.Filter{Focus: &c.Editor, Name: key.NameEscape},
			key.Filter{Focus: &c.Editor, Name: key.NameUpArrow},
			key.Filter{Focus: &c.Editor, Name: key.NameDownArrow},
		)
		if !ok {
			break
		}
		if e, ok := ev.(key.Event); ok && e.State == key.Press {
			switch e.Name {
			case key.NameEscape:
				result.Closed = true
			case key.NameUpArrow:
				c.MoveSelection(-1)
			case key.NameDownArrow:
				c.MoveSelection(1)
			}
		}
	}

	// Process editor internal events
	for {
		ev, ok := c.Editor.Update(gtx)
		if !ok {
			break
		}
		if _, ok := ev.(widget.SubmitEvent); ok {
			if len(c.Filtered) > 0 && c.ActiveIndex >= 0 && c.ActiveIndex < len(c.Filtered) {
				result.Action = c.Filtered[c.ActiveIndex].Action
			}
			c.Editor.SetText("")
			result.Submitted = true
			result.Closed = true
		}
	}

	// Filter logic
	txt := c.Editor.Text()
	if txt != c.lastText {
		c.lastText = txt
		c.Filtered = nil
		q := strings.ToLower(txt)
		for _, item := range c.Items {
			if strings.Contains(strings.ToLower(item.Name), q) {
				c.Filtered = append(c.Filtered, item)
			}
		}
		c.ActiveIndex = 0
		c.List.ScrollTo(0)
	}

	// Overlay background
	bg := ColorBg
	bg.A = 200

	w := gtx.Constraints.Max.X
	h := gtx.Constraints.Max.Y

	paint.FillShape(gtx.Ops, bg, clip.Rect{Max: image.Pt(w, h)}.Op())

	// Container size
	boxW := 480
	if boxW > w {
		boxW = w - 40
	}
	if boxW < 100 {
		boxW = 100
	}

	x := (w - boxW) / 2
	y := h / 5

	off := op.Offset(image.Pt(x, y)).Push(gtx.Ops)

	gtx.Constraints.Min.X = boxW
	gtx.Constraints.Max.X = boxW

	maxH := h * 2 / 3
	if maxH < 300 {
		maxH = 300
	}
	gtx.Constraints.Max.Y = maxH

	// Draw main container background
	containerBg := ColorTitleBar
	containerRect := image.Rectangle{Max: image.Pt(boxW, maxH)}
	containerRR := clip.UniformRRect(containerRect, gtx.Dp(4))
	paint.FillShape(gtx.Ops, containerBg, containerRR.Op(gtx.Ops))

	// Clip content to container
	cl := containerRR.Push(gtx.Ops)

	macro := op.Record(gtx.Ops)
	dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Search Input Area
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Top: unit.Dp(12), Bottom: unit.Dp(12),
				Left: unit.Dp(16), Right: unit.Dp(16),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, unit.Sp(16), "⌘")
						lbl.Color = blendColor(ColorText, -80)
						return layout.Inset{Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return lbl.Layout(gtx)
						})
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						ed := material.Editor(th, &c.Editor, "")
						ed.Color = ColorText
						ed.HintColor = blendColor(ColorText, -120)
						ed.Font = font.Font{Typeface: "Geist Mono, monospace"}
						ed.TextSize = unit.Sp(14)
						return ed.Layout(gtx)
					}),
				)
			})
		}),
		// Separator Line
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			sepColor := blendColor(ColorTitleBar, 20)
			rect := image.Rectangle{
				Min: image.Pt(16, 0),
				Max: image.Pt(boxW-16, gtx.Dp(1)),
			}
			paint.FillShape(gtx.Ops, sepColor, clip.Rect(rect).Op())
			return layout.Dimensions{Size: image.Pt(boxW, gtx.Dp(1))}
		}),
		// Command List Area
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if len(c.Filtered) == 0 {
				return layout.Inset{
					Top: unit.Dp(20), Bottom: unit.Dp(20),
					Left: unit.Dp(16), Right: unit.Dp(16),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, unit.Sp(12), "No commands found")
					lbl.Color = blendColor(ColorText, -100)
					lbl.Font.Typeface = "Geist Mono, monospace"
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return lbl.Layout(gtx)
					})
				})
			}
			return layout.Inset{
				Top: unit.Dp(6), Bottom: unit.Dp(6),
				Left: unit.Dp(6), Right: unit.Dp(6),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.List(th, &c.List).Layout(gtx, len(c.Filtered), func(gtx layout.Context, index int) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					item := c.Filtered[index]

					return layout.Inset{Top: unit.Dp(1), Bottom: unit.Dp(1)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Stack{}.Layout(gtx,
							layout.Expanded(func(gtx layout.Context) layout.Dimensions {
								if index == c.ActiveIndex {
									hoverBg := ColorTabHoverBg
									rr := clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, gtx.Dp(3))
									paint.FillShape(gtx.Ops, hoverBg, rr.Op(gtx.Ops))
								}
								return layout.Dimensions{Size: gtx.Constraints.Min}
							}),
							layout.Stacked(func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{
									Top: unit.Dp(8), Bottom: unit.Dp(8),
									Left: unit.Dp(10), Right: unit.Dp(10),
								}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
										// Name
										layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
											lbl := material.Label(th, unit.Sp(13), item.Name)
											lbl.Font.Typeface = "Geist Mono, monospace"
											if index == c.ActiveIndex {
												lbl.Color = ColorText
											} else {
												lbl.Color = blendColor(ColorText, -50)
											}
											return lbl.Layout(gtx)
										}),
										// Shortcut
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											if item.Shortcut == "" {
												return layout.Dimensions{}
											}
											return layout.Inset{Left: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												return layout.Stack{}.Layout(gtx,
													layout.Expanded(func(gtx layout.Context) layout.Dimensions {
														bg := blendColor(ColorTitleBar, 15)
														rr := clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, gtx.Dp(2))
														paint.FillShape(gtx.Ops, bg, rr.Op(gtx.Ops))
														return layout.Dimensions{Size: gtx.Constraints.Min}
													}),
													layout.Stacked(func(gtx layout.Context) layout.Dimensions {
														return layout.Inset{
															Top: unit.Dp(3), Bottom: unit.Dp(3),
															Left: unit.Dp(6), Right: unit.Dp(6),
														}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
															lbl := material.Label(th, unit.Sp(10), item.Shortcut)
															lbl.Color = blendColor(ColorText, -80)
															lbl.Font.Typeface = "Geist Mono, monospace"
															return lbl.Layout(gtx)
														})
													}),
												)
											})
										}),
									)
								})
							}),
						)
					})
				})
			})
		}),
	)
	call := macro.Stop()

	call.Add(gtx.Ops)
	cl.Pop()

	// Draw border
	borderColor := color.NRGBA{R: 30, G: 30, B: 30, A: 100}
	paint.FillShape(gtx.Ops, borderColor, clip.Stroke{
		Path:  containerRR.Path(gtx.Ops),
		Width: float32(gtx.Dp(0.5)),
	}.Op())

	off.Pop()

	_ = dims
	return layout.Dimensions{Size: image.Pt(w, h)}, result
}

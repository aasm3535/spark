package components

import (
	"image"
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
	Name   string
	Action config.Action
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
		{"New Tab", config.ActionNewTab},
		{"Close Tab", config.ActionCloseTab},
		{"Next Tab", config.ActionNextTab},
		{"Previous Tab", config.ActionPrevTab},
		{"Scroll Up", config.ActionScrollUp},
		{"Scroll Down", config.ActionScrollDown},
		{"Scroll Page Up", config.ActionScrollPageUp},
		{"Scroll Page Down", config.ActionScrollPageDown},
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
	boxW := 400
	if boxW > w {
		boxW = w - 40 // simple padding
	}
	if boxW < 100 {
		boxW = 100
	}

	x := (w - boxW) / 2
	y := h / 6

	off := op.Offset(image.Pt(x, y)).Push(gtx.Ops)

	gtx.Constraints.Min.X = boxW
	gtx.Constraints.Max.X = boxW

	maxH := h * 2 / 3
	if maxH < 200 {
		maxH = 200
	}
	gtx.Constraints.Max.Y = maxH

	macro := op.Record(gtx.Ops)
	dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Search Input Area
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				ed := material.Editor(th, &c.Editor, "Type a command...")
				ed.Color = ColorText
				ed.HintColor = blendColor(ColorText, -100)
				ed.Font = font.Font{Typeface: "Segoe UI, sans-serif"}
				ed.TextSize = unit.Sp(14)
				return ed.Layout(gtx)
			})
		}),
		// Separator Line
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			rect := image.Rectangle{Max: image.Pt(gtx.Constraints.Max.X, gtx.Dp(1))}
			paint.FillShape(gtx.Ops, blendColor(ColorTitleBar, 20), clip.Rect(rect).Op())
			return layout.Dimensions{Size: rect.Max}
		}),
		// Command List Area
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if len(c.Filtered) == 0 {
				return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, unit.Sp(13), "No commands found.")
					lbl.Color = blendColor(ColorText, -80)
					return lbl.Layout(gtx)
				})
			}
			return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.List(th, &c.List).Layout(gtx, len(c.Filtered), func(gtx layout.Context, index int) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					item := c.Filtered[index]

					return layout.UniformInset(unit.Dp(2)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Stack{}.Layout(gtx,
							layout.Expanded(func(gtx layout.Context) layout.Dimensions {
								if index == c.ActiveIndex {
									rr := clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, gtx.Dp(6))
									paint.FillShape(gtx.Ops, ColorTabHoverBg, rr.Op(gtx.Ops))
								}
								return layout.Dimensions{Size: gtx.Constraints.Min}
							}),
							layout.Stacked(func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{
									Top: unit.Dp(4), Bottom: unit.Dp(4),
									Left: unit.Dp(8), Right: unit.Dp(8),
								}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									lbl := material.Label(th, unit.Sp(13), item.Name)
									lbl.Font.Typeface = "Segoe UI, sans-serif"
									if index == c.ActiveIndex {
										lbl.Color = ColorText
										lbl.Font.Weight = font.Bold
									} else {
										lbl.Color = blendColor(ColorText, -40)
									}
									return lbl.Layout(gtx)
								})
							}),
						)
					})
				})
			})
		}),
	)
	call := macro.Stop()

	// Draw Background Box with rounded corners
	rect := image.Rectangle{Max: dims.Size}
	rr := clip.UniformRRect(rect, gtx.Dp(8))

	paint.FillShape(gtx.Ops, ColorTitleBar, rr.Op(gtx.Ops))

	cl := rr.Push(gtx.Ops)
	call.Add(gtx.Ops)
	cl.Pop()

	// Draw Border
	paint.FillShape(gtx.Ops, blendColor(ColorTitleBar, 30), clip.Stroke{
		Path:  rr.Path(gtx.Ops),
		Width: float32(gtx.Dp(1)),
	}.Op())

	off.Pop()

	return layout.Dimensions{Size: image.Pt(w, h)}, result
}

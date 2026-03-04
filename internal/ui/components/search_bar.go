package components

import (
	"image"

	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type SearchBar struct {
	Editor   widget.Editor
	focused  bool
	lastText string
}

type SearchBarResult struct {
	QueryChanged bool
	Query        string
	Next         bool
	Prev         bool
	Closed       bool
}

func (s *SearchBar) Layout(
	gtx layout.Context,
	th *material.Theme,
	active bool,
) (layout.Dimensions, SearchBarResult) {
	var result SearchBarResult

	if !active {
		s.focused = false
		return layout.Dimensions{}, result
	}

	s.Editor.SingleLine = true
	s.Editor.Submit = true

	// Key interception
	for {
		ev, ok := gtx.Event(
			key.Filter{Focus: &s.Editor, Name: key.NameEscape},
			key.Filter{Focus: &s.Editor, Name: key.NameReturn},
			key.Filter{Focus: &s.Editor, Name: key.NameEnter},
			key.Filter{Focus: &s.Editor, Name: key.NameReturn, Required: key.ModShift},
			key.Filter{Focus: &s.Editor, Name: key.NameEnter, Required: key.ModShift},
		)
		if !ok {
			break
		}
		if e, ok := ev.(key.Event); ok && e.State == key.Press {
			if e.Name == key.NameEscape {
				result.Closed = true
			} else if e.Name == key.NameEnter || e.Name == key.NameReturn {
				if e.Modifiers == key.ModShift {
					result.Prev = true
				} else {
					result.Next = true
				}
			}
		}
	}

	for {
		ev, ok := s.Editor.Update(gtx)
		if !ok {
			break
		}
		if _, ok := ev.(widget.SubmitEvent); ok {
			result.Next = true
		}
	}

	txt := s.Editor.Text()
	if txt != s.lastText {
		s.lastText = txt
		result.QueryChanged = true
		result.Query = txt
	}

	if !s.focused {
		gtx.Execute(key.FocusCmd{Tag: &s.Editor})
		s.focused = true
	}

	// Layout the search bar at the top right
	width := 300
	height := 40

	if width > gtx.Constraints.Max.X {
		width = gtx.Constraints.Max.X
	}

	x := gtx.Constraints.Max.X - width - 20
	y := 10

	off := op.Offset(image.Pt(x, y)).Push(gtx.Ops)

	gtx.Constraints.Min = image.Pt(width, height)
	gtx.Constraints.Max = image.Pt(width, height)

	macro := op.Record(gtx.Ops)

	layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		ed := material.Editor(th, &s.Editor, "Find...")
		ed.Color = ColorText
		ed.HintColor = blendColor(ColorText, -100)
		ed.Font = font.Font{Typeface: "Segoe UI, sans-serif"}
		return ed.Layout(gtx)
	})

	call := macro.Stop()

	rect := image.Rectangle{Max: image.Pt(width, height)}
	rr := clip.UniformRRect(rect, gtx.Dp(6))

	paint.FillShape(gtx.Ops, ColorTitleBar, rr.Op(gtx.Ops))

	cl := rr.Push(gtx.Ops)
	call.Add(gtx.Ops)
	cl.Pop()

	paint.FillShape(gtx.Ops, blendColor(ColorTitleBar, 30), clip.Stroke{
		Path:  rr.Path(gtx.Ops),
		Width: float32(gtx.Dp(1)),
	}.Op())

	off.Pop()

	// Return zero dimensions to let it overlay without affecting layout
	return layout.Dimensions{Size: image.Pt(0, 0)}, result
}

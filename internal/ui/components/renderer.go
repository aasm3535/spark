package components

import (
	"image"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"yutug.lol/spark/internal/terminal"
)

// Renderer draws a terminal snapshot into the current Gio frame.
type Renderer struct {
	CellWidth  int
	CellHeight int
}

// Layout renders all visible cells from snap and returns the total dimensions.
func (r *Renderer) Layout(gtx layout.Context, th *material.Theme, snap terminal.Snapshot) layout.Dimensions {
	fontSize := th.TextSize
	monoFont := font.Font{Typeface: th.Face, Weight: font.Light}

	// ── Measure a single cell ─────────────────────────────────────────────
	lbl := widget.Label{MaxLines: 1}
	macro := op.Record(gtx.Ops)
	measureGtx := gtx
	measureGtx.Constraints.Min = image.Point{}
	charDims := lbl.Layout(measureGtx, th.Shaper, monoFont, fontSize, "X", op.CallOp{})
	macro.Stop()

	spPx := float32(gtx.Sp(fontSize))
	r.CellWidth = max1(charDims.Size.X, 7)
	r.CellHeight = max1(int(spPx*1.35), 14)

	cw := r.CellWidth
	ch := r.CellHeight

	maxW := gtx.Constraints.Max.X
	maxH := gtx.Constraints.Max.Y

	visCols := clamp1(maxW/cw, 1, snap.Cols)
	visRows := clamp1(maxH/ch, 1, snap.Rows)

	// ── Render each visible cell ──────────────────────────────────────────
	for row := 0; row < visRows && row < len(snap.Screen); row++ {
		line := snap.Screen[row]
		for col := 0; col < visCols && col < len(line); col++ {
			cell := line[col]
			x := col * cw
			y := row * ch

			cellRect := image.Rectangle{
				Min: image.Pt(x, y),
				Max: image.Pt(x+cw, y+ch),
			}

			// Search Highlight
			isSearchMatch := false
			if highlights, ok := snap.SearchMatches[row]; ok {
				for _, h := range highlights {
					if col >= h.StartCol && col < h.EndCol {
						isSearchMatch = true
						break
					}
				}
			}

			// Background
			bg := ResolveColor(cell.Bg, terminal.ColorBg)
			if isSearchMatch {
				// Highlight search matches with an orange/yellow background
				paint.FillShape(gtx.Ops, blendColor(ColorTitleBar, 60), clip.Rect(cellRect).Op())
			} else if bg != terminal.ColorBg {
				paint.FillShape(gtx.Ops, bg, clip.Rect(cellRect).Op())
			}

			// Cursor underline.
			if snap.ShowCursor && row == snap.CurY && col == snap.CurX {
				cursorRect := image.Rectangle{
					Min: image.Pt(x, y+ch-2),
					Max: image.Pt(x+cw, y+ch),
				}
				paint.FillShape(gtx.Ops, ColorCursor, clip.Rect(cursorRect).Op())
			}

			// Glyph — skip blank cells.
			if cell.Ch != 0 && cell.Ch != ' ' {
				r.drawGlyph(gtx, th, monoFont, fontSize, cell, x, y, cw, ch)
			}
		}
	}

	totalW := min1(visCols*cw, maxW)
	totalH := min1(visRows*ch, maxH)
	return layout.Dimensions{Size: image.Pt(totalW, totalH)}
}

// ColsRows returns how many columns and rows fit in the given pixel area.
func (r *Renderer) ColsRows(widthPx, heightPx int) (cols, rows int) {
	if r.CellWidth == 0 || r.CellHeight == 0 {
		return terminal.DefaultCols, terminal.DefaultRows
	}
	cols = clamp1(widthPx/r.CellWidth, 20, 512)
	rows = clamp1(heightPx/r.CellHeight, 5, 200)
	return
}

// ─── Glyph drawing ────────────────────────────────────────────────────────────

func (r *Renderer) drawGlyph(
	gtx layout.Context,
	th *material.Theme,
	base font.Font,
	size unit.Sp,
	cell terminal.Cell,
	x, y, cw, ch int,
) {
	off := op.Offset(image.Pt(x, y)).Push(gtx.Ops)

	// Give the label up to 2 cell-widths so wide glyphs are not clipped.
	gtx2 := gtx
	gtx2.Constraints = layout.Exact(image.Pt(cw*2, ch))

	fnt := base
	if cell.Bold {
		fnt.Weight = font.Bold
	}

	fg := ResolveColor(cell.Fg, ColorText)

	macro := op.Record(gtx2.Ops)
	paint.ColorOp{Color: fg}.Add(gtx2.Ops)
	colorCall := macro.Stop()

	lbl := widget.Label{MaxLines: 1}
	lbl.Layout(gtx2, th.Shaper, fnt, size, string(cell.Ch), colorCall)

	off.Pop()
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func max1(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min1(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clamp1(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

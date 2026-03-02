package ui

import (
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
	"yutug.lol/spark/internal/terminal"
)

// Renderer draws the terminal screen snapshot into the current Gio frame.
type Renderer struct {
	// CellWidth / CellHeight are recomputed each frame from the font size.
	CellWidth  int
	CellHeight int
}

// Layout renders the terminal snapshot and returns its dimensions.
func (r *Renderer) Layout(gtx layout.Context, th *material.Theme, snap terminal.Snapshot) layout.Dimensions {
	fontSize := th.TextSize
	monoFont := font.Font{Typeface: th.Face, Weight: font.Light}

	// ── cell size estimation ───────────────────────────────────────────────
	lbl := widget.Label{MaxLines: 1}
	macro := op.Record(gtx.Ops)
	measureGtx := gtx
	measureGtx.Constraints.Min = image.Point{}
	dims := lbl.Layout(measureGtx, th.Shaper, monoFont, fontSize, "X", op.CallOp{})
	macro.Stop()

	spPx := float32(gtx.Sp(fontSize))
	r.CellWidth = max1(dims.Size.X, 7)
	r.CellHeight = max1(int(spPx*1.35), 14)

	cw := r.CellWidth
	ch := r.CellHeight

	maxW := gtx.Constraints.Max.X
	maxH := gtx.Constraints.Max.Y

	visCols := clamp1(maxW/cw, 1, snap.Cols)
	visRows := clamp1(maxH/ch, 1, snap.Rows)

	// ── render each visible cell ───────────────────────────────────────────
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

			// ── background ────────────────────────────────────────────────
			bg := resolveColor(cell.Bg, terminal.ColorBg)
			if bg != terminal.ColorBg {
				paint.FillShape(gtx.Ops, bg, clip.Rect(cellRect).Op())
			}

			// ── cursor underline ──────────────────────────────────────────
			if snap.ShowCursor && row == snap.CurY && col == snap.CurX {
				cursorRect := image.Rectangle{
					Min: image.Pt(x, y+ch-2),
					Max: image.Pt(x+cw, y+ch),
				}
				paint.FillShape(gtx.Ops, ColorCursor, clip.Rect(cursorRect).Op())
			}

			// ── glyph ─────────────────────────────────────────────────────
			if cell.Ch != 0 && cell.Ch != ' ' {
				drawGlyph(gtx, th, monoFont, fontSize, cell, x, y, cw, ch)
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

// ── glyph drawing ─────────────────────────────────────────────────────────────

func drawGlyph(
	gtx layout.Context,
	th *material.Theme,
	base font.Font,
	size unit.Sp,
	cell terminal.Cell,
	x, y, cw, ch int,
) {
	off := op.Offset(image.Pt(x, y)).Push(gtx.Ops)

	// give the label up to 2 cell-widths so wide glyphs are not clipped
	gtx2 := gtx
	gtx2.Constraints = layout.Exact(image.Pt(cw*2, ch))

	fnt := base
	if cell.Bold {
		fnt.Weight = font.Bold
	}

	fg := resolveColor(cell.Fg, ColorText)

	// record the colour op so widget.Label can consume it
	macro := op.Record(gtx2.Ops)
	paint.ColorOp{Color: fg}.Add(gtx2.Ops)
	colorCall := macro.Stop()

	lbl := widget.Label{MaxLines: 1}
	lbl.Layout(gtx2, th.Shaper, fnt, size, string(cell.Ch), colorCall)

	off.Pop()
}

// ── helpers ───────────────────────────────────────────────────────────────────

func resolveColor(c, fallback color.NRGBA) color.NRGBA {
	if c == (color.NRGBA{}) {
		return fallback
	}
	return c
}

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

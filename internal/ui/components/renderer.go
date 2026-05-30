package components

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/mattn/go-runewidth"
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

			isSearchMatch := false
			if highlights, ok := snap.SearchMatches[row]; ok {
				for _, h := range highlights {
					if col >= h.StartCol && col < h.EndCol {
						isSearchMatch = true
						break
					}
				}
			}

			isSelected := false
			if snap.HasSelection {
				isSelected = isCellSelected(snap.SelStart, snap.SelEnd, row, col, snap.ScrollOffset)
			}

			bg := cell.Bg
			if bg == terminal.ColorBg || bg == (color.NRGBA{}) {
				bg = ColorBg
			}
			if isSelected {
				paint.FillShape(gtx.Ops, color.NRGBA{R: 50, G: 100, B: 200, A: 120}, clip.Rect(cellRect).Op())
			} else if isSearchMatch {
				paint.FillShape(gtx.Ops, blendColor(ColorTitleBar, 60), clip.Rect(cellRect).Op())
			} else {
				paint.FillShape(gtx.Ops, bg, clip.Rect(cellRect).Op())
			}

			// Cursor — reverse video
			if snap.ShowCursor && row == snap.CurY && col == snap.CurX {
				fg := ResolveColor(cell.Fg, ColorText)
				curBg := bg
				paint.FillShape(gtx.Ops, fg, clip.Rect(cellRect).Op())
				cell = terminal.Cell{
					Ch:   cell.Ch,
					Fg:   curBg,
					Bg:   fg,
					Bold: cell.Bold,
				}
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

type boxStyle struct {
	left, right, top, bottom uint8 // 0: none, 1: light, 2: heavy, 3: double
}

var boxStyles = [128]boxStyle{
	// 2500 - 2507
	0x00: {1, 1, 0, 0}, // ─
	0x01: {2, 2, 0, 0}, // ━
	0x02: {0, 0, 1, 1}, // │
	0x03: {0, 0, 2, 2}, // ┃
	0x04: {1, 1, 0, 0}, // ┄
	0x05: {2, 2, 0, 0}, // ┅
	0x06: {1, 1, 0, 0}, // ┈
	0x07: {2, 2, 0, 0}, // ┉
	// 2508 - 250F
	0x08: {0, 0, 1, 1}, // ┊
	0x09: {0, 0, 2, 2}, // ┋
	0x0a: {0, 0, 1, 1}, // ┈
	0x0b: {0, 0, 2, 2}, // ┉
	0x0c: {0, 1, 0, 1}, // ┌
	0x0d: {0, 2, 0, 1}, // ┍
	0x0e: {0, 1, 0, 2}, // ┎
	0x0f: {0, 2, 0, 2}, // ┏
	// 2510 - 2517
	0x10: {1, 0, 0, 1}, // ┐
	0x11: {2, 0, 0, 1}, // ┑
	0x12: {1, 0, 0, 2}, // ┒
	0x13: {2, 0, 0, 2}, // ┓
	0x14: {0, 1, 1, 0}, // └
	0x15: {0, 2, 1, 0}, // ┕
	0x16: {0, 1, 2, 0}, // ┖
	0x17: {0, 2, 2, 0}, // ┗
	// 2518 - 251F
	0x18: {1, 0, 1, 0}, // ┘
	0x19: {2, 0, 1, 0}, // ┙
	0x1a: {1, 0, 2, 0}, // ┚
	0x1b: {2, 0, 2, 0}, // ┛
	0x1c: {0, 1, 1, 1}, // ├
	0x1d: {0, 2, 1, 1}, // ┝
	0x1e: {0, 1, 2, 1}, // ┞
	0x1f: {0, 1, 1, 2}, // ┟
	// 2520 - 2527
	0x20: {0, 1, 2, 2}, // ┠
	0x21: {0, 2, 1, 1}, // ┡
	0x22: {0, 2, 2, 1}, // ┢
	0x23: {0, 2, 2, 2}, // ┣
	0x24: {1, 0, 1, 1}, // ┤
	0x25: {2, 0, 1, 1}, // ┥
	0x26: {1, 0, 2, 1}, // ┦
	0x27: {1, 0, 1, 2}, // ┧
	// 2528 - 252F
	0x28: {1, 0, 2, 2}, // ┨
	0x29: {2, 0, 1, 1}, // ┩
	0x2a: {2, 0, 2, 1}, // ┪
	0x2b: {2, 0, 2, 2}, // ┫
	0x2c: {1, 1, 0, 1}, // ┬
	0x2d: {2, 1, 0, 1}, // ┭
	0x2e: {1, 2, 0, 1}, // ┮
	0x2f: {2, 2, 0, 1}, // ┯
	// 2530 - 2537
	0x30: {1, 1, 0, 2}, // ┰
	0x31: {2, 1, 0, 2}, // ┱
	0x32: {1, 2, 0, 2}, // ┲
	0x33: {2, 2, 0, 2}, // ┳
	0x34: {1, 1, 1, 0}, // ┴
	0x35: {2, 1, 1, 0}, // ┵
	0x36: {1, 2, 1, 0}, // ┶
	0x37: {2, 2, 1, 0}, // ┷
	// 2538 - 253F
	0x38: {1, 1, 2, 0}, // ┸
	0x39: {2, 1, 2, 0}, // ┹
	0x3a: {1, 2, 2, 0}, // ┺
	0x3b: {2, 2, 2, 0}, // ┻
	0x3c: {1, 1, 1, 1}, // ┼
	0x3d: {2, 1, 1, 1}, // ┽
	0x3e: {1, 2, 1, 1}, // ┾
	0x3f: {2, 2, 1, 1}, // ┿
	// 2540 - 2547
	0x40: {1, 1, 2, 1}, // ╀
	0x41: {1, 1, 1, 2}, // ╁
	0x42: {1, 1, 2, 2}, // ╂
	0x43: {2, 1, 2, 1}, // ╃
	0x44: {1, 2, 2, 1}, // ╄
	0x45: {2, 1, 1, 2}, // ╅
	0x46: {1, 2, 1, 2}, // ╆
	0x47: {2, 2, 2, 1}, // ╇
	// 2548 - 254F
	0x48: {2, 2, 1, 2}, // ╈
	0x49: {2, 1, 2, 2}, // ╉
	0x4a: {1, 2, 2, 2}, // ╊
	0x4b: {2, 2, 2, 2}, // ╋
	// Double lines
	0x50: {3, 3, 0, 0}, // ═
	0x51: {0, 0, 3, 3}, // ║
	0x52: {0, 3, 0, 1}, // ╒
	0x53: {0, 1, 0, 3}, // ╓
	0x54: {0, 3, 0, 3}, // ╔
	0x55: {3, 0, 0, 1}, // ╕
	0x56: {1, 0, 0, 3}, // ╖
	0x57: {3, 0, 0, 3}, // ╗
	0x58: {0, 3, 1, 0}, // ╘
	0x59: {0, 1, 3, 0}, // ╙
	0x5a: {0, 3, 3, 0}, // ╚
	0x5b: {3, 0, 1, 0}, // ╛
	0x5c: {1, 0, 3, 0}, // ╜
	0x5d: {3, 0, 3, 0}, // ╝
	0x5e: {0, 1, 3, 3}, // ╞
	0x5f: {0, 3, 1, 1}, // ╟
	0x60: {0, 3, 3, 3}, // ╠
	0x61: {1, 0, 3, 3}, // ╡
	0x62: {3, 0, 1, 1}, // ╢
	0x63: {3, 0, 3, 3}, // ╣
	0x64: {3, 3, 0, 1}, // ╤
	0x65: {1, 1, 0, 3}, // ╥
	0x66: {3, 3, 0, 3}, // ╦
	0x67: {3, 3, 1, 0}, // ╧
	0x68: {1, 1, 3, 0}, // ╨
	0x69: {3, 3, 3, 0}, // ╩
	0x6a: {1, 1, 3, 3}, // ╪
	0x6b: {3, 3, 1, 1}, // ╫
	0x6c: {3, 3, 3, 3}, // ╬
	// Rounded (arcs, drawn as straight corners for alignment)
	0x6d: {0, 1, 0, 1}, // ╭
	0x6e: {1, 0, 0, 1}, // ╮
	0x6f: {1, 0, 1, 0}, // ╯
	0x70: {0, 1, 1, 0}, // ╰
	// Diagonals (drawn manually)
	0x71: {0, 0, 0, 0}, // ╱
	0x72: {0, 0, 0, 0}, // ╲
	0x73: {0, 0, 0, 0}, // ╳
	// Single ended
	0x74: {1, 0, 0, 0}, // ╴
	0x75: {0, 0, 1, 0}, // ╵
	0x76: {0, 1, 0, 0}, // ╶
	0x77: {0, 0, 0, 1}, // ╷
	0x78: {2, 0, 0, 0}, // ╸
	0x79: {0, 0, 2, 0}, // ╹
	0x7a: {0, 2, 0, 0}, // ╺
	0x7b: {0, 0, 0, 2}, // ╻
	0x7c: {1, 2, 0, 0}, // ╼
	0x7d: {0, 0, 1, 2}, // ╽
	0x7e: {2, 1, 0, 0}, // ╾
	0x7f: {0, 0, 2, 1}, // ╿
}

func drawDiagonals(gtx layout.Context, ch rune, col color.NRGBA, cw, chH int) {
	t := max1(1, chH/14)

	drawLine := func(x1, y1, x2, y2 float32) {
		var path clip.Path
		path.Begin(gtx.Ops)

		dx := x2 - x1
		dy := y2 - y1
		length := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		if length == 0 {
			return
		}

		nx := -dy / length * float32(t) / 2
		ny := dx / length * float32(t) / 2

		path.MoveTo(f32.Pt(x1+nx, y1+ny))
		path.LineTo(f32.Pt(x2+nx, y2+ny))
		path.LineTo(f32.Pt(x2-nx, y2-ny))
		path.LineTo(f32.Pt(x1-nx, y1-ny))
		path.Close()

		paint.FillShape(gtx.Ops, col, clip.Outline{Path: path.End()}.Op())
	}

	switch ch {
	case 0x2571: // ╱
		drawLine(float32(cw), 0, 0, float32(chH))
	case 0x2572: // ╲
		drawLine(0, 0, float32(cw), float32(chH))
	case 0x2573: // ╳
		drawLine(float32(cw), 0, 0, float32(chH))
		drawLine(0, 0, float32(cw), float32(chH))
	}
}

func drawBoxDrawing(gtx layout.Context, ch rune, col color.NRGBA, cw, chH int) bool {
	if ch < 0x2500 || ch > 0x257F {
		return false
	}

	if ch >= 0x2571 && ch <= 0x2573 {
		drawDiagonals(gtx, ch, col, cw, chH)
		return true
	}

	style := boxStyles[ch-0x2500]

	tLight := max1(1, chH/14)
	tHeavy := tLight * 2

	tD := tLight
	sD := max1(2, tLight*2)

	midX := cw / 2
	midY := chH / 2

	drawRect := func(x1, y1, x2, y2 int) {
		if x1 < 0 {
			x1 = 0
		}
		if y1 < 0 {
			y1 = 0
		}
		if x2 > cw {
			x2 = cw
		}
		if y2 > chH {
			y2 = chH
		}
		if x1 >= x2 || y1 >= y2 {
			return
		}
		paint.FillShape(gtx.Ops, col, clip.Rect(image.Rect(x1, y1, x2, y2)).Op())
	}

	// 1. LEFT
	switch style.left {
	case 1:
		yMin := midY - tLight/2
		drawRect(0, yMin, midX+tLight, yMin+tLight)
	case 2:
		yMin := midY - tHeavy/2
		drawRect(0, yMin, midX+tHeavy, yMin+tHeavy)
	case 3:
		yMin1 := midY - sD/2 - tD
		yMin2 := midY + sD/2
		drawRect(0, yMin1, midX+sD/2+tD, yMin1+tD)
		drawRect(0, yMin2, midX+sD/2+tD, yMin2+tD)
	}

	// 2. RIGHT
	switch style.right {
	case 1:
		yMin := midY - tLight/2
		drawRect(midX-tLight/2, yMin, cw, yMin+tLight)
	case 2:
		yMin := midY - tHeavy/2
		drawRect(midX-tHeavy/2, yMin, cw, yMin+tHeavy)
	case 3:
		yMin1 := midY - sD/2 - tD
		yMin2 := midY + sD/2
		drawRect(midX-sD/2-tD, yMin1, cw, yMin1+tD)
		drawRect(midX-sD/2-tD, yMin2, cw, yMin2+tD)
	}

	// 3. TOP
	switch style.top {
	case 1:
		xMin := midX - tLight/2
		drawRect(xMin, 0, xMin+tLight, midY+tLight)
	case 2:
		xMin := midX - tHeavy/2
		drawRect(xMin, 0, xMin+tHeavy, midY+tHeavy)
	case 3:
		xMin1 := midX - sD/2 - tD
		xMin2 := midX + sD/2
		drawRect(xMin1, 0, xMin1+tD, midY+sD/2+tD)
		drawRect(xMin2, 0, xMin2+tD, midY+sD/2+tD)
	}

	// 4. BOTTOM
	switch style.bottom {
	case 1:
		xMin := midX - tLight/2
		drawRect(xMin, midY-tLight/2, xMin+tLight, chH)
	case 2:
		xMin := midX - tHeavy/2
		drawRect(xMin, midY-tHeavy/2, xMin+tHeavy, chH)
	case 3:
		xMin1 := midX - sD/2 - tD
		xMin2 := midX + sD/2
		drawRect(xMin1, midY-sD/2-tD, xMin1+tD, chH)
		drawRect(xMin2, midY-sD/2-tD, xMin2+tD, chH)
	}

	return true
}

func drawBlockElement(gtx layout.Context, ch rune, col color.NRGBA, cw, chH int) bool {
	drawRect := func(x1, y1, x2, y2 int) {
		if x1 < 0 {
			x1 = 0
		}
		if y1 < 0 {
			y1 = 0
		}
		if x2 > cw {
			x2 = cw
		}
		if y2 > chH {
			y2 = chH
		}
		if x1 >= x2 || y1 >= y2 {
			return
		}
		paint.FillShape(gtx.Ops, col, clip.Rect(image.Rect(x1, y1, x2, y2)).Op())
	}

	drawRectAlpha := func(x1, y1, x2, y2 int, alpha uint8) {
		if x1 < 0 {
			x1 = 0
		}
		if y1 < 0 {
			y1 = 0
		}
		if x2 > cw {
			x2 = cw
		}
		if y2 > chH {
			y2 = chH
		}
		if x1 >= x2 || y1 >= y2 {
			return
		}
		c := col
		c.A = uint8(uint32(c.A) * uint32(alpha) / 255)
		paint.FillShape(gtx.Ops, c, clip.Rect(image.Rect(x1, y1, x2, y2)).Op())
	}

	switch ch {
	case 0x2580: // ▀
		drawRect(0, 0, cw, max1(1, chH/2))
	case 0x2581:
		drawRect(0, chH-max1(1, chH/8), cw, chH)
	case 0x2582:
		drawRect(0, chH-max1(1, chH/4), cw, chH)
	case 0x2583:
		drawRect(0, chH-max1(1, chH*3/8), cw, chH)
	case 0x2584: // ▄
		drawRect(0, max1(1, chH/2), cw, chH)
	case 0x2585:
		drawRect(0, chH-max1(1, chH*5/8), cw, chH)
	case 0x2586:
		drawRect(0, chH-max1(1, chH*3/4), cw, chH)
	case 0x2587:
		drawRect(0, chH-max1(1, chH*7/8), cw, chH)
	case 0x2588: // █
		drawRect(0, 0, cw, chH)
	case 0x2589:
		drawRect(0, 0, max1(1, cw*7/8), chH)
	case 0x258A:
		drawRect(0, 0, max1(1, cw*3/4), chH)
	case 0x258B:
		drawRect(0, 0, max1(1, cw*5/8), chH)
	case 0x258C: // ▌
		drawRect(0, 0, max1(1, cw/2), chH)
	case 0x258D:
		drawRect(0, 0, max1(1, cw*3/8), chH)
	case 0x258E:
		drawRect(0, 0, max1(1, cw/4), chH)
	case 0x258F:
		drawRect(0, 0, max1(1, cw/8), chH)
	case 0x2590: // ▐
		drawRect(max1(1, cw/2), 0, cw, chH)
	case 0x2591: // ░
		drawRectAlpha(0, 0, cw, chH, 64)
	case 0x2592: // ▒
		drawRectAlpha(0, 0, cw, chH, 128)
	case 0x2593: // ▓
		drawRectAlpha(0, 0, cw, chH, 192)
	case 0x2594:
		drawRect(0, 0, cw, max1(1, chH/8))
	case 0x2595:
		drawRect(cw-max1(1, cw/8), 0, cw, chH)
	case 0x2596:
		drawRect(0, max1(1, chH/2), max1(1, cw/2), chH)
	case 0x2597:
		drawRect(max1(1, cw/2), max1(1, chH/2), cw, chH)
	case 0x2598:
		drawRect(0, 0, max1(1, cw/2), max1(1, chH/2))
	case 0x2599:
		drawRect(0, 0, max1(1, cw/2), chH)
		drawRect(max1(1, cw/2), max1(1, chH/2), cw, chH)
	case 0x259A:
		drawRect(0, 0, max1(1, cw/2), max1(1, chH/2))
		drawRect(max1(1, cw/2), max1(1, chH/2), cw, chH)
	case 0x259B:
		drawRect(0, 0, cw, max1(1, chH/2))
		drawRect(0, max1(1, chH/2), max1(1, cw/2), chH)
	case 0x259C:
		drawRect(0, 0, cw, max1(1, chH/2))
		drawRect(max1(1, cw/2), max1(1, chH/2), cw, chH)
	case 0x259D:
		drawRect(max1(1, cw/2), 0, cw, max1(1, chH/2))
	case 0x259E:
		drawRect(max1(1, cw/2), 0, cw, max1(1, chH/2))
		drawRect(0, max1(1, chH/2), max1(1, cw/2), chH)
	case 0x259F:
		drawRect(max1(1, cw/2), 0, cw, chH)
		drawRect(0, max1(1, chH/2), max1(1, cw/2), chH)
	default:
		return false
	}
	return true
}

func drawBraillePattern(gtx layout.Context, ch rune, col color.NRGBA, cw, chH int) bool {
	if ch < 0x2800 || ch > 0x28FF {
		return false
	}

	offset := int(ch - 0x2800)

	leftX := cw / 3
	rightX := cw - cw/3

	y1 := chH / 5
	y2 := chH * 2 / 5
	y3 := chH * 3 / 5
	y4 := chH * 4 / 5

	dotSize := max1(2, (chH+6)/10)

	drawDot := func(x, y int) {
		x1 := x - dotSize/2
		x2 := x1 + dotSize
		y1_coord := y - dotSize/2
		y2_coord := y1_coord + dotSize

		if x1 < 0 {
			x1 = 0
		}
		if y1_coord < 0 {
			y1_coord = 0
		}
		if x2 > cw {
			x2 = cw
		}
		if y2_coord > chH {
			y2_coord = chH
		}
		if x1 >= x2 || y1_coord >= y2_coord {
			return
		}

		paint.FillShape(gtx.Ops, col, clip.Rect(image.Rect(x1, y1_coord, x2, y2_coord)).Op())
	}

	if offset&0x01 != 0 {
		drawDot(leftX, y1)
	}
	if offset&0x02 != 0 {
		drawDot(leftX, y2)
	}
	if offset&0x04 != 0 {
		drawDot(leftX, y3)
	}
	if offset&0x08 != 0 {
		drawDot(rightX, y1)
	}
	if offset&0x10 != 0 {
		drawDot(rightX, y2)
	}
	if offset&0x20 != 0 {
		drawDot(rightX, y3)
	}
	if offset&0x40 != 0 {
		drawDot(leftX, y4)
	}
	if offset&0x80 != 0 {
		drawDot(rightX, y4)
	}

	return true
}

func drawGeometricShape(gtx layout.Context, ch rune, col color.NRGBA, cw, chH int) bool {
	switch ch {
	case 0x25B6, 0x25B8, 0x25C0, 0x25C2, 0x25B2, 0x25B4, 0x25BC, 0x25BE:
	default:
		return false
	}

	cx := float32(cw) / 2
	cy := float32(chH) / 2

	var p clip.Path
	p.Begin(gtx.Ops)

	switch ch {
	// Large right-pointing triangle ▶
	case 0x25B6:
		size := float32(min1(cw, chH)) * 0.5
		p.MoveTo(f32.Pt(cx-size*0.5, cy-size*0.6))
		p.LineTo(f32.Pt(cx+size*0.7, cy))
		p.LineTo(f32.Pt(cx-size*0.5, cy+size*0.6))
		p.Close()
		paint.FillShape(gtx.Ops, col, clip.Outline{Path: p.End()}.Op())
		return true

	// Small right-pointing triangle ▸
	case 0x25B8:
		size := float32(min1(cw, chH)) * 0.35
		p.MoveTo(f32.Pt(cx-size*0.5, cy-size*0.6))
		p.LineTo(f32.Pt(cx+size*0.7, cy))
		p.LineTo(f32.Pt(cx-size*0.5, cy+size*0.6))
		p.Close()
		paint.FillShape(gtx.Ops, col, clip.Outline{Path: p.End()}.Op())
		return true

	// Large left-pointing triangle ◀
	case 0x25C0:
		size := float32(min1(cw, chH)) * 0.5
		p.MoveTo(f32.Pt(cx+size*0.5, cy-size*0.6))
		p.LineTo(f32.Pt(cx-size*0.7, cy))
		p.LineTo(f32.Pt(cx+size*0.5, cy+size*0.6))
		p.Close()
		paint.FillShape(gtx.Ops, col, clip.Outline{Path: p.End()}.Op())
		return true

	// Small left-pointing triangle ◂
	case 0x25C2:
		size := float32(min1(cw, chH)) * 0.35
		p.MoveTo(f32.Pt(cx+size*0.5, cy-size*0.6))
		p.LineTo(f32.Pt(cx-size*0.7, cy))
		p.LineTo(f32.Pt(cx+size*0.5, cy+size*0.6))
		p.Close()
		paint.FillShape(gtx.Ops, col, clip.Outline{Path: p.End()}.Op())
		return true

	// Large up-pointing triangle ▲
	case 0x25B2:
		size := float32(min1(cw, chH)) * 0.5
		p.MoveTo(f32.Pt(cx-size*0.6, cy+size*0.5))
		p.LineTo(f32.Pt(cx, cy-size*0.7))
		p.LineTo(f32.Pt(cx+size*0.6, cy+size*0.5))
		p.Close()
		paint.FillShape(gtx.Ops, col, clip.Outline{Path: p.End()}.Op())
		return true

	// Small up-pointing triangle ▴
	case 0x25B4:
		size := float32(min1(cw, chH)) * 0.35
		p.MoveTo(f32.Pt(cx-size*0.6, cy+size*0.5))
		p.LineTo(f32.Pt(cx, cy-size*0.7))
		p.LineTo(f32.Pt(cx+size*0.6, cy+size*0.5))
		p.Close()
		paint.FillShape(gtx.Ops, col, clip.Outline{Path: p.End()}.Op())
		return true

	// Large down-pointing triangle ▼
	case 0x25BC:
		size := float32(min1(cw, chH)) * 0.5
		p.MoveTo(f32.Pt(cx-size*0.6, cy-size*0.5))
		p.LineTo(f32.Pt(cx, cy+size*0.7))
		p.LineTo(f32.Pt(cx+size*0.6, cy-size*0.5))
		p.Close()
		paint.FillShape(gtx.Ops, col, clip.Outline{Path: p.End()}.Op())
		return true

	// Small down-pointing triangle ▾
	case 0x25BE:
		size := float32(min1(cw, chH)) * 0.35
		p.MoveTo(f32.Pt(cx-size*0.6, cy-size*0.5))
		p.LineTo(f32.Pt(cx, cy+size*0.7))
		p.LineTo(f32.Pt(cx+size*0.6, cy-size*0.5))
		p.Close()
		paint.FillShape(gtx.Ops, col, clip.Outline{Path: p.End()}.Op())
		return true
	}

	return false
}

func drawBoxOrBlock(gtx layout.Context, ch rune, col color.NRGBA, cw, chH int) bool {
	if ch >= 0x2500 && ch <= 0x257F {
		return drawBoxDrawing(gtx, ch, col, cw, chH)
	}
	if ch >= 0x2580 && ch <= 0x259F {
		return drawBlockElement(gtx, ch, col, cw, chH)
	}
	if ch >= 0x2800 && ch <= 0x28FF {
		return drawBraillePattern(gtx, ch, col, cw, chH)
	}
	if ch >= 0x25A0 && ch <= 0x25FF {
		return drawGeometricShape(gtx, ch, col, cw, chH)
	}
	return false
}

func (r *Renderer) drawGlyph(
	gtx layout.Context,
	th *material.Theme,
	base font.Font,
	size unit.Sp,
	cell terminal.Cell,
	x, y, cw, ch int,
) {
	fg := ResolveColor(cell.Fg, ColorText)

	// Record all drawing ops in a macro so we can apply offset + clip after.
	macro := op.Record(gtx.Ops)

	if drawBoxOrBlock(gtx, cell.Ch, fg, cw, ch) {
		call := macro.Stop()
		off := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
		call.Add(gtx.Ops)
		off.Pop()
		return
	}

	gtx2 := gtx
	gtx2.Constraints = layout.Exact(image.Pt(cw*2, ch))

	fnt := base
	if cell.Bold {
		fnt.Weight = font.Bold
	}

	// Record color op (no push active).
	colorMacro := op.Record(gtx2.Ops)
	paint.ColorOp{Color: fg}.Add(gtx2.Ops)
	colorCall := colorMacro.Stop()

	// Record label layout (no push active).
	lbl := widget.Label{MaxLines: 1}
	lbl.Layout(gtx2, th.Shaper, fnt, size, string(cell.Ch), colorCall)

	call := macro.Stop()

	clipWidth := cw
	if runewidth.RuneWidth(cell.Ch) == 2 {
		clipWidth = cw * 2
	}

	// Apply offset, clip to cell bounds, then replay the recorded ops.
	off := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
	cl := clip.Rect(image.Rectangle{Max: image.Pt(clipWidth, ch)}).Push(gtx.Ops)
	call.Add(gtx.Ops)
	cl.Pop()
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

func isCellSelected(start, end terminal.SelPos, row, col, scrollOffset int) bool {
	if start.Row > end.Row || (start.Row == end.Row && start.Col > end.Col) {
		start, end = end, start
	}

	absRow := row - scrollOffset

	if absRow < start.Row || absRow > end.Row {
		return false
	}

	if start.Row == end.Row {
		return absRow == start.Row && col >= start.Col && col < end.Col
	}

	if absRow == start.Row {
		return col >= start.Col
	}
	if absRow == end.Row {
		return col < end.Col
	}

	return true
}

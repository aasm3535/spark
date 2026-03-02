package terminal

import "image/color"

// Palette ────────────────────────────────────────────────────────────────────

var (
	ColorBg   = color.NRGBA{R: 18, G: 18, B: 24, A: 255}
	ColorText = color.NRGBA{R: 220, G: 220, B: 230, A: 255}

	// ANSI 16-colour table (normal + bright)
	AnsiColors = [16]color.NRGBA{
		{R: 30, G: 30, B: 30, A: 255},    // 0  black
		{R: 205, G: 49, B: 49, A: 255},   // 1  red
		{R: 13, G: 188, B: 121, A: 255},  // 2  green
		{R: 229, G: 229, B: 16, A: 255},  // 3  yellow
		{R: 36, G: 114, B: 200, A: 255},  // 4  blue
		{R: 188, G: 63, B: 188, A: 255},  // 5  magenta
		{R: 17, G: 168, B: 205, A: 255},  // 6  cyan
		{R: 200, G: 200, B: 200, A: 255}, // 7  white
		{R: 102, G: 102, B: 102, A: 255}, // 8  bright black
		{R: 241, G: 76, B: 76, A: 255},   // 9  bright red
		{R: 35, G: 209, B: 139, A: 255},  // 10 bright green
		{R: 245, G: 245, B: 67, A: 255},  // 11 bright yellow
		{R: 59, G: 142, B: 234, A: 255},  // 12 bright blue
		{R: 214, G: 112, B: 214, A: 255}, // 13 bright magenta
		{R: 41, G: 184, B: 219, A: 255},  // 14 bright cyan
		{R: 255, G: 255, B: 255, A: 255}, // 15 bright white
	}
)

// Cell ────────────────────────────────────────────────────────────────────────

// Cell represents a single character cell on the terminal screen.
type Cell struct {
	Ch   rune
	Fg   color.NRGBA
	Bg   color.NRGBA
	Bold bool
}

// DefaultCell returns a blank cell with the default colors.
func DefaultCell() Cell {
	return Cell{Ch: ' ', Fg: ColorText, Bg: ColorBg}
}

// Ansi256 converts an xterm-256 color index to NRGBA.
func Ansi256(idx int) color.NRGBA {
	if idx < 0 {
		idx = 0
	}
	if idx > 255 {
		idx = 255
	}
	if idx < 16 {
		return AnsiColors[idx]
	}
	if idx >= 232 {
		v := uint8((idx-232)*10 + 8)
		return color.NRGBA{R: v, G: v, B: v, A: 255}
	}
	idx -= 16
	b := idx % 6
	g := (idx / 6) % 6
	r := idx / 36
	scale := func(v int) uint8 {
		if v == 0 {
			return 0
		}
		return uint8(55 + v*40)
	}
	return color.NRGBA{R: scale(r), G: scale(g), B: scale(b), A: 255}
}

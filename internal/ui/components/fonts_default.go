//go:build !windows

package components

import "gioui.org/font"

func loadSystemFonts(cfgFontFamily string) []font.FontFace {
	return nil
}

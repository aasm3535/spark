package components

import (
	"image/color"

	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/font/opentype"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"yutug.lol/spark/internal/assets"
	"yutug.lol/spark/internal/config"
)

// ─── Palette ──────────────────────────────────────────────────────────────────

var (
	ColorBg        = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	ColorTitleBar  = color.NRGBA{R: 10, G: 10, B: 10, A: 255}
	ColorTitleText = color.NRGBA{R: 161, G: 161, B: 161, A: 255}
	ColorText      = color.NRGBA{R: 237, G: 237, B: 237, A: 255}
	ColorCursor    = color.NRGBA{R: 255, G: 255, B: 255, A: 255}

	ColorBtnHoverClose   = color.NRGBA{R: 225, G: 29, B: 72, A: 255}
	ColorBtnHoverNeutral = color.NRGBA{R: 255, G: 255, B: 255, A: 15}

	ColorTabActiveBg   = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	ColorTabInactiveBg = color.NRGBA{R: 10, G: 10, B: 10, A: 255}
	ColorTabHoverBg    = color.NRGBA{R: 26, G: 26, B: 26, A: 255}
)

// blendColor lightens (amount > 0) or darkens (amount < 0) a colour.
func blendColor(c color.NRGBA, amount int) color.NRGBA {
	clamp := func(v int) uint8 {
		if v < 0 {
			return 0
		}
		if v > 255 {
			return 255
		}
		return uint8(v)
	}
	return color.NRGBA{
		R: clamp(int(c.R) + amount),
		G: clamp(int(c.G) + amount),
		B: clamp(int(c.B) + amount),
		A: c.A,
	}
}

// ResolveColor returns fallback when c is the zero value.
func ResolveColor(c, fallback color.NRGBA) color.NRGBA {
	if c == (color.NRGBA{}) {
		return fallback
	}
	return c
}

// applyThemeColors updates the palette from a loaded config.
func applyThemeColors(cfg *config.Config) {
	if cfg == nil || cfg.CustomTheme == nil {
		ColorTabActiveBg = ColorBg
		ColorTabInactiveBg = blendColor(ColorTitleBar, -4)
		ColorTabHoverBg = blendColor(ColorBg, 10)
		return
	}

	t := cfg.CustomTheme

	if c, err := config.ParseHexColor(t.Bg); err == nil && t.Bg != "" {
		ColorBg = c
	}
	if c, err := config.ParseHexColor(t.Fg); err == nil && t.Fg != "" {
		ColorText = c
	}
	if c, err := config.ParseHexColor(t.TitleBar); err == nil && t.TitleBar != "" {
		ColorTitleBar = c
	}
	if c, err := config.ParseHexColor(t.TitleText); err == nil && t.TitleText != "" {
		ColorTitleText = c
	}
	if c, err := config.ParseHexColor(t.Cursor); err == nil && t.Cursor != "" {
		ColorCursor = c
	}
	if c, err := config.ParseHexColor(t.BtnHoverClose); err == nil && t.BtnHoverClose != "" {
		ColorBtnHoverClose = c
	}
	if c, err := config.ParseHexColor(t.BtnHoverNeutral); err == nil && t.BtnHoverNeutral != "" {
		ColorBtnHoverNeutral = c
	}

	// Derive tab colours from final palette.
	ColorTabActiveBg = ColorBg
	ColorTabInactiveBg = blendColor(ColorTitleBar, -4)
	ColorTabHoverBg = blendColor(ColorBg, 10)

	// Override with explicit values if provided.
	if c, err := config.ParseHexColor(t.TabActiveBg); err == nil && t.TabActiveBg != "" {
		ColorTabActiveBg = c
	}
	if c, err := config.ParseHexColor(t.TabInactiveBg); err == nil && t.TabInactiveBg != "" {
		ColorTabInactiveBg = c
	}
	if c, err := config.ParseHexColor(t.TabHoverBg); err == nil && t.TabHoverBg != "" {
		ColorTabHoverBg = c
	}
}

// ─── Font descriptors ─────────────────────────────────────────────────────────

const MonoFaceList = "Geist Mono"

type fontFile struct {
	data   []byte
	weight font.Weight
	style  font.Style
}

var geistMonoFiles = []fontFile{
	{assets.GeistMonoLight, font.Light, font.Regular},
	{assets.GeistMonoRegular, font.Normal, font.Regular},
	{assets.GeistMonoMedium, font.Medium, font.Regular},
	{assets.GeistMonoBold, font.Bold, font.Regular},
}

// ─── Theme constructor ────────────────────────────────────────────────────────

// NewTheme builds a material.Theme with the Geist Mono font and applies
// colours from cfg.
func NewTheme(cfg *config.Config) *material.Theme {
	applyThemeColors(cfg)

	collection := loadGeistMono()
	collection = append(collection, gofont.Collection()...)
	if cfg != nil {
		collection = append(collection, loadSystemFonts(cfg.FontFamily)...)
	} else {
		collection = append(collection, loadSystemFonts("")...)
	}

	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(collection))

	fallbackList := ", Segoe UI Emoji, Apple Color Emoji, Noto Color Emoji, DejaVu Sans, Segoe UI Symbol, sans-serif"

	if cfg != nil && cfg.FontFamily != "" {
		th.Face = font.Typeface(cfg.FontFamily + fallbackList)
	} else {
		th.Face = font.Typeface(MonoFaceList + fallbackList)
	}

	if cfg != nil && cfg.FontSize > 0 {
		th.TextSize = unit.Sp(float32(cfg.FontSize))
	} else {
		th.TextSize = unit.Sp(14)
	}

	th.Palette.Bg = ColorBg
	th.Palette.Fg = ColorText
	return th
}

func loadGeistMono() []font.FontFace {
	var faces []font.FontFace
	for _, f := range geistMonoFiles {
		parsed, err := opentype.Parse(f.data)
		if err != nil {
			continue
		}
		faces = append(faces, font.FontFace{
			Font: font.Font{
				Typeface: "Geist Mono",
				Weight:   f.weight,
				Style:    f.style,
			},
			Face: parsed,
		})
	}
	return faces
}

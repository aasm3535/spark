package components

import (
	"image/color"
	"log"

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
	ColorBg        = color.NRGBA{R: 18, G: 18, B: 24, A: 255}
	ColorTitleBar  = color.NRGBA{R: 22, G: 22, B: 30, A: 255}
	ColorTitleText = color.NRGBA{R: 160, G: 160, B: 180, A: 255}
	ColorText      = color.NRGBA{R: 220, G: 220, B: 230, A: 255}
	ColorCursor    = color.NRGBA{R: 130, G: 200, B: 255, A: 220}

	ColorBtnHoverClose   = color.NRGBA{R: 196, G: 43, B: 28, A: 255}
	ColorBtnHoverNeutral = color.NRGBA{R: 255, G: 255, B: 255, A: 18}

	ColorTabActiveBg   = color.NRGBA{R: 18, G: 18, B: 24, A: 255}
	ColorTabInactiveBg = color.NRGBA{R: 22, G: 22, B: 30, A: 255}
	ColorTabHoverBg    = color.NRGBA{R: 30, G: 30, B: 40, A: 255}
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

const MonoFaceList = "Iosevka Fixed, Go Mono, monospace"

type iosevkaFile struct {
	data   []byte
	weight font.Weight
	style  font.Style
}

var iosevkaFiles = []iosevkaFile{
	{assets.IosevkaLight, font.Light, font.Regular},
	{assets.IosevkaMedium, font.Medium, font.Regular},
	{assets.IosevkaThin, font.Thin, font.Regular},
}

// ─── Theme constructor ────────────────────────────────────────────────────────

// NewTheme builds a material.Theme with the Iosevka Fixed font and applies
// colours from cfg.
func NewTheme(cfg *config.Config) *material.Theme {
	applyThemeColors(cfg)

	collection := loadIosevka()
	collection = append(collection, gofont.Collection()...)

	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(collection))

	if cfg != nil && cfg.FontFamily != "" {
		th.Face = font.Typeface(cfg.FontFamily)
	} else {
		th.Face = font.Typeface(MonoFaceList)
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

func loadIosevka() []font.FontFace {
	var faces []font.FontFace
	for _, f := range iosevkaFiles {
		parsed, err := opentype.ParseCollection(f.data)
		if err != nil {
			log.Printf("components: parsing Iosevka variant: %v", err)
			continue
		}
		for _, face := range parsed {
			faces = append(faces, font.FontFace{
				Font: font.Font{
					Typeface: "Iosevka Fixed",
					Weight:   f.weight,
					Style:    f.style,
				},
				Face: face.Face,
			})
		}
	}
	if len(faces) > 0 {
		log.Printf("components: loaded %d Iosevka Fixed face(s)", len(faces))
	} else {
		log.Printf("components: Iosevka Fixed not loaded, falling back to Go Mono")
	}
	return faces
}

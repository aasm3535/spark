package config

import (
	"fmt"
	"image/color"
	"strconv"
)

// ParseHexColor parses a hex string like "#RRGGBB", "#RRGGBBAA", or "#RGB" into color.NRGBA.
func ParseHexColor(s string) (color.NRGBA, error) {
	if len(s) > 0 && s[0] == '#' {
		s = s[1:]
	}

	var r, g, b, a uint8
	a = 255 // Default alpha

	switch len(s) {
	case 6:
		parsed, err := strconv.ParseUint(s, 16, 32)
		if err != nil {
			return color.NRGBA{}, err
		}
		r = uint8(parsed >> 16)
		g = uint8((parsed >> 8) & 0xFF)
		b = uint8(parsed & 0xFF)
	case 8:
		parsed, err := strconv.ParseUint(s, 16, 32)
		if err != nil {
			return color.NRGBA{}, err
		}
		r = uint8(parsed >> 24)
		g = uint8((parsed >> 16) & 0xFF)
		b = uint8((parsed >> 8) & 0xFF)
		a = uint8(parsed & 0xFF)
	case 3:
		parsed, err := strconv.ParseUint(s, 16, 32)
		if err != nil {
			return color.NRGBA{}, err
		}
		r = uint8((parsed >> 8) * 17) // * 17 is equivalent to repeating the hex digit (e.g., F -> FF)
		g = uint8(((parsed >> 4) & 0xF) * 17)
		b = uint8((parsed & 0xF) * 17)
	default:
		return color.NRGBA{}, fmt.Errorf("invalid color format: %s", s)
	}

	return color.NRGBA{R: r, G: g, B: b, A: a}, nil
}

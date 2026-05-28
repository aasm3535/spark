//go:build !windows

package components

import (
	"os"
	"path/filepath"
	"strings"

	"gioui.org/font"
	"gioui.org/font/opentype"
)

func loadSystemFonts(cfgFontFamily string) []font.FontFace {
	var faces []font.FontFace

	// Load user configured font if specified (except default Geist Mono which is already embedded)
	if cfgFontFamily != "" && !strings.Contains(strings.ToLower(cfgFontFamily), "geist mono") {
		// If multiple font families are listed in a comma-separated fallback list, load all of them
		families := strings.Split(cfgFontFamily, ",")
		for _, fam := range families {
			fam = strings.TrimSpace(fam)
			if fam != "" {
				faces = append(faces, loadSystemFont(fam)...)
			}
		}
	}

	// Always load standard macOS/Linux fallback fonts for Emojis and symbols
	faces = append(faces, loadSystemFont("Apple Color Emoji")...)
	faces = append(faces, loadSystemFont("Noto Color Emoji")...)
	faces = append(faces, loadSystemFont("DejaVu Sans")...)

	return faces
}

// loadSystemFont searches for a font by family name in common macOS and Linux directories and parses it.
func loadSystemFont(familyName string) []font.FontFace {
	var files []string
	familyLower := strings.ToLower(familyName)

	// macOS standard font directories
	macDirs := []string{
		"/System/Library/Fonts",
		"/Library/Fonts",
	}

	// Linux standard font directories
	linuxDirs := []string{
		"/usr/share/fonts",
		"/usr/local/share/fonts",
	}
	if home, err := os.UserHomeDir(); err == nil {
		macDirs = append(macDirs, filepath.Join(home, "Library", "Fonts"))
		linuxDirs = append(linuxDirs, filepath.Join(home, ".local", "share", "fonts"))
		linuxDirs = append(linuxDirs, filepath.Join(home, ".fonts"))
	}

	// Combine dirs based on OS or scan all since they won't exist on the wrong OS
	searchDirs := append(macDirs, linuxDirs...)

	for _, dir := range searchDirs {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".ttf" || ext == ".otf" || ext == ".ttc" || ext == ".otc" {
				base := strings.ToLower(filepath.Base(path))
				// Check if the filename contains the familyName
				// e.g. "Apple Color Emoji.ttc" contains "apple color emoji"
				// "NotoColorEmoji.ttf" contains "noto color emoji" (ignoring spaces)
				normalizedName := strings.ReplaceAll(strings.ToLower(familyName), " ", "")
				normalizedBase := strings.ReplaceAll(base, " ", "")
				normalizedBase = strings.ReplaceAll(normalizedBase, "-", "")
				normalizedBase = strings.ReplaceAll(normalizedBase, "_", "")

				if strings.Contains(normalizedBase, normalizedName) {
					files = append(files, path)
				}
			}
			return nil
		})
	}

	var faces []font.FontFace
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		facesInFile, err := opentype.ParseCollection(data)
		if err != nil {
			continue
		}
		for _, f := range facesInFile {
			f.Font.Typeface = font.Typeface(familyName)
			faces = append(faces, f)
		}
	}
	return faces
}

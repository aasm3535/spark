//go:build windows

package components

import (
	"os"
	"path/filepath"
	"strings"

	"gioui.org/font"
	"gioui.org/font/opentype"
	"golang.org/x/sys/windows/registry"
)

func findSystemFontFiles(familyName string) []string {
	if familyName == "" {
		return nil
	}
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`, registry.READ)
	if err != nil {
		return nil
	}
	defer k.Close()

	var files []string
	familyLower := strings.ToLower(familyName)

	userFontsDir := ""
	if home, err := os.UserHomeDir(); err == nil {
		userFontsDir = filepath.Join(home, "AppData", "Local", "Microsoft", "Windows", "Fonts")
	}

	checkKey := func(rootKey registry.Key, defaultDir string) {
		k2, err := registry.OpenKey(rootKey, `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`, registry.READ)
		if err != nil {
			return
		}
		defer k2.Close()

		valNames, err := k2.ReadValueNames(0)
		if err != nil {
			return
		}

		for _, name := range valNames {
			if strings.Contains(strings.ToLower(name), familyLower) {
				file, _, err := k2.GetStringValue(name)
				if err == nil && file != "" {
					var fullPath string
					if filepath.IsAbs(file) {
						fullPath = file
					} else {
						fullPath = filepath.Join(defaultDir, file)
					}
					files = append(files, fullPath)
				}
			}
		}
	}

	windir := os.Getenv("WINDIR")
	if windir == "" {
		windir = "C:\\Windows"
	}
	systemFontsDir := filepath.Join(windir, "Fonts")

	checkKey(registry.LOCAL_MACHINE, systemFontsDir)
	if userFontsDir != "" {
		checkKey(registry.CURRENT_USER, userFontsDir)
	}

	return files
}

func loadSystemFont(familyName string) []font.FontFace {
	files := findSystemFontFiles(familyName)
	var faces []font.FontFace
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		parsed, err := opentype.Parse(data)
		if err != nil {
			continue
		}

		weight := font.Normal
		style := font.Regular
		fileLower := strings.ToLower(filepath.Base(file))
		if strings.Contains(fileLower, "bold") || strings.Contains(fileLower, "-b") || strings.Contains(fileLower, "_b") {
			weight = font.Bold
		} else if strings.Contains(fileLower, "light") || strings.Contains(fileLower, "-l") {
			weight = font.Light
		} else if strings.Contains(fileLower, "medium") || strings.Contains(fileLower, "-m") {
			weight = font.Medium
		}

		if strings.Contains(fileLower, "italic") || strings.Contains(fileLower, "oblique") {
			style = font.Italic
		}

		faces = append(faces, font.FontFace{
			Font: font.Font{
				Typeface: font.Typeface(familyName),
				Weight:   weight,
				Style:    style,
			},
			Face: parsed,
		})
	}
	return faces
}

func findNerdFontFamilies() []string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`, registry.READ)
	if err != nil {
		return nil
	}
	defer k.Close()

	names, err := k.ReadValueNames(0)
	if err != nil {
		return nil
	}

	var families []string
	seen := make(map[string]bool)

	checkNames := func(valNames []string) {
		for _, name := range valNames {
			if strings.Contains(strings.ToLower(name), "nerd font") {
				clean := name
				if pos := strings.Index(name, "("); pos != -1 {
					clean = name[:pos]
				}
				clean = strings.TrimSpace(clean)
				for _, w := range []string{"Regular", "Bold", "Italic", "Light", "Medium", "Oblique", "Mono"} {
					clean = strings.TrimSuffix(clean, " "+w)
					clean = strings.TrimSuffix(clean, w)
				}
				clean = strings.TrimSpace(clean)
				if clean != "" && !seen[clean] {
					seen[clean] = true
					families = append(families, clean)
				}
			}
		}
	}

	checkNames(names)

	userKey := `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`
	if k2, err := registry.OpenKey(registry.CURRENT_USER, userKey, registry.READ); err == nil {
		if names2, err := k2.ReadValueNames(0); err == nil {
			checkNames(names2)
		}
		k2.Close()
	}

	return families
}

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

	// Always load standard Windows fallback fonts for Emojis and symbols
	var fallbackFaces []font.FontFace
	fallbackFaces = append(fallbackFaces, loadSystemFont("Segoe UI Emoji")...)
	fallbackFaces = append(fallbackFaces, loadSystemFont("Segoe UI Symbol")...)
	fallbackFaces = append(fallbackFaces, loadSystemFont("Segoe MDL2 Assets")...)

	// Automatically detect and load any Nerd Fonts installed on the system as fallback
	nerdFamilies := findNerdFontFamilies()
	for _, fam := range nerdFamilies {
		fallbackFaces = append(fallbackFaces, loadSystemFont(fam)...)
	}

	// 1. Add the fallback fonts under their original family names
	faces = append(faces, fallbackFaces...)

	// 2. Add copies of the fallback fonts mapped to target font families (Geist Mono and user's config font).
	// This registers them as valid fallback faces under the active font family names, allowing
	// Gio's text shaper to query them when a character/glyph is missing in the base font.
	targetTypefaces := []string{"Geist Mono"}
	if cfgFontFamily != "" && !strings.Contains(strings.ToLower(cfgFontFamily), "geist mono") {
		families := strings.Split(cfgFontFamily, ",")
		for _, fam := range families {
			fam = strings.TrimSpace(fam)
			if fam != "" {
				targetTypefaces = append(targetTypefaces, fam)
			}
		}
	}

	for _, target := range targetTypefaces {
		for _, fbFace := range fallbackFaces {
			for _, w := range []font.Weight{font.Light, font.Normal, font.Medium, font.Bold} {
				for _, s := range []font.Style{font.Regular, font.Italic} {
					faceCopy := fbFace
					faceCopy.Font.Typeface = font.Typeface(target)
					faceCopy.Font.Weight = w
					faceCopy.Font.Style = s
					faces = append(faces, faceCopy)
				}
			}
		}
	}

	return faces
}

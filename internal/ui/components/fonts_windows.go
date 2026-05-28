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
	faces = append(faces, loadSystemFont("Segoe UI Emoji")...)
	faces = append(faces, loadSystemFont("Segoe UI Symbol")...)
	faces = append(faces, loadSystemFont("Segoe MDL2 Assets")...)

	// Automatically detect and load any Nerd Fonts installed on the system as fallback
	nerdFamilies := findNerdFontFamilies()
	for _, fam := range nerdFamilies {
		faces = append(faces, loadSystemFont(fam)...)
	}

	return faces
}

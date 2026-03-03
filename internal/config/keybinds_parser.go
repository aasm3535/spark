package config

import (
	"strings"

	"gioui.org/io/key"
)

// Chord represents a parsed key combination: a set of required modifiers
// and a key name.
type Chord struct {
	Mods key.Modifiers
	Name key.Name
}

// ParseChord parses a human-readable chord string such as "Ctrl+Shift+T" or
// "Shift+PageUp" into a Chord.
//
// Rules:
//   - Tokens are split on "+".
//   - Modifier tokens (case-insensitive): ctrl, shift, alt.
//   - The last non-modifier token is the key name.
//   - Special key names are mapped to Gio key.Name constants.
//   - An empty string or a string with no key name returns the zero Chord and
//     ok=false.
func ParseChord(s string) (Chord, bool) {
	if s == "" {
		return Chord{}, false
	}

	tokens := strings.Split(s, "+")

	var mods key.Modifiers
	var namePart string

	for _, tok := range tokens {
		switch strings.ToLower(strings.TrimSpace(tok)) {
		case "ctrl":
			mods |= key.ModCtrl
		case "shift":
			mods |= key.ModShift
		case "alt":
			mods |= key.ModAlt
		default:
			namePart = strings.TrimSpace(tok)
		}
	}

	if namePart == "" {
		return Chord{}, false
	}

	name := resolveKeyName(namePart)
	return Chord{Mods: mods, Name: name}, true
}

// Matches reports whether the given key.Event matches this chord.
// Only key.Press events are considered a match.
func (c Chord) Matches(e key.Event) bool {
	if e.State != key.Press {
		return false
	}
	return e.Modifiers == c.Mods && e.Name == c.Name
}

// String returns the chord in the canonical "Mod+Mod+Key" form.
func (c Chord) String() string {
	var parts []string
	if c.Mods.Contain(key.ModCtrl) {
		parts = append(parts, "Ctrl")
	}
	if c.Mods.Contain(key.ModShift) {
		parts = append(parts, "Shift")
	}
	if c.Mods.Contain(key.ModAlt) {
		parts = append(parts, "Alt")
	}
	parts = append(parts, string(c.Name))
	return strings.Join(parts, "+")
}

// resolveKeyName maps human-friendly key name strings to Gio key.Name values.
// Single upper-case letters are passed through as-is.
func resolveKeyName(s string) key.Name {
	switch strings.ToLower(s) {
	// Navigation
	case "pageup", "pgup":
		return key.NamePageUp
	case "pagedown", "pgdn", "pgdown":
		return key.NamePageDown
	case "uparrow", "up":
		return key.NameUpArrow
	case "downarrow", "down":
		return key.NameDownArrow
	case "leftarrow", "left":
		return key.NameLeftArrow
	case "rightarrow", "right":
		return key.NameRightArrow
	case "home":
		return key.NameHome
	case "end":
		return key.NameEnd

	// Editing
	case "return", "enter":
		return key.NameReturn
	case "backspace", "deletebackward":
		return key.NameDeleteBackward
	case "delete", "deleteforward":
		return key.NameDeleteForward
	case "tab":
		return key.NameTab
	case "space":
		return key.NameSpace
	case "escape", "esc":
		return key.NameEscape

	// Function keys
	case "f1":
		return "F1"
	case "f2":
		return "F2"
	case "f3":
		return "F3"
	case "f4":
		return "F4"
	case "f5":
		return "F5"
	case "f6":
		return "F6"
	case "f7":
		return "F7"
	case "f8":
		return "F8"
	case "f9":
		return "F9"
	case "f10":
		return "F10"
	case "f11":
		return "F11"
	case "f12":
		return "F12"
	}

	// Single letter — normalise to upper-case as Gio expects.
	upper := strings.ToUpper(s)
	if len(upper) == 1 {
		return key.Name(upper)
	}

	// Fall back: return as-is and let Gio sort it out.
	return key.Name(s)
}

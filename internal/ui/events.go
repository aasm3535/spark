package ui

import (
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"yutug.lol/spark/internal/terminal"
)

// buildFilters returns the full list of input event filters for the terminal.
func buildFilters(tag *struct{}) []event.Filter {
	ctrl := key.ModCtrl
	shift := key.ModShift
	alt := key.ModAlt

	f := []event.Filter{
		pointer.Filter{
			Target:  tag,
			Kinds:   pointer.Scroll,
			ScrollY: pointer.ScrollRange{Min: -1000000, Max: 1000000},
		},
		key.FocusFilter{Target: tag},
		key.Filter{Focus: tag, Name: "", Optional: ctrl | shift | alt},
		key.Filter{Focus: tag, Name: key.NameReturn},
		key.Filter{Focus: tag, Name: key.NameEnter},
		key.Filter{Focus: tag, Name: key.NameDeleteBackward},
		key.Filter{Focus: tag, Name: key.NameDeleteForward},
		key.Filter{Focus: tag, Name: key.NameUpArrow},
		key.Filter{Focus: tag, Name: key.NameUpArrow, Required: shift},
		key.Filter{Focus: tag, Name: key.NameDownArrow},
		key.Filter{Focus: tag, Name: key.NameDownArrow, Required: shift},
		key.Filter{Focus: tag, Name: key.NameLeftArrow},
		key.Filter{Focus: tag, Name: key.NameRightArrow},
		key.Filter{Focus: tag, Name: key.NameHome},
		key.Filter{Focus: tag, Name: key.NameEnd},
		key.Filter{Focus: tag, Name: key.NamePageUp},
		key.Filter{Focus: tag, Name: key.NamePageUp, Required: shift},
		key.Filter{Focus: tag, Name: key.NamePageUp, Required: ctrl},
		key.Filter{Focus: tag, Name: key.NamePageDown},
		key.Filter{Focus: tag, Name: key.NamePageDown, Required: shift},
		key.Filter{Focus: tag, Name: key.NamePageDown, Required: ctrl},
		key.Filter{Focus: tag, Name: key.NameEscape},
		key.Filter{Focus: tag, Name: key.NameTab},
		key.Filter{Focus: tag, Name: key.NameTab, Required: shift},
		key.Filter{Focus: tag, Name: key.NameSpace},
		key.Filter{Focus: tag, Name: "\\", Required: ctrl},
		key.Filter{Focus: tag, Name: "]", Required: ctrl},
		key.Filter{Focus: tag, Name: "[", Required: ctrl},
		key.Filter{Focus: tag, Name: "-", Required: ctrl},
		key.Filter{Focus: tag, Name: "F1"},
		key.Filter{Focus: tag, Name: "F2"},
		key.Filter{Focus: tag, Name: "F3"},
		key.Filter{Focus: tag, Name: "F4"},
		key.Filter{Focus: tag, Name: "F5"},
		key.Filter{Focus: tag, Name: "F6"},
		key.Filter{Focus: tag, Name: "F7"},
		key.Filter{Focus: tag, Name: "F8"},
		key.Filter{Focus: tag, Name: "F9"},
		key.Filter{Focus: tag, Name: "F10"},
		key.Filter{Focus: tag, Name: "F11"},
		key.Filter{Focus: tag, Name: "F12"},
	}

	// Ctrl+A–Z; T and W also accept Shift (tab management binds)
	for _, l := range []string{
		"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M",
		"N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z",
	} {
		switch l {
		case "T", "W":
			f = append(f, key.Filter{Focus: tag, Name: key.Name(l), Required: ctrl, Optional: shift})
		default:
			f = append(f, key.Filter{Focus: tag, Name: key.Name(l), Required: ctrl})
		}
	}

	return f
}

// handleEvents processes all pending input events for the current frame.
func (win *Window) handleEvents(gtx layout.Context) {
	tag := &win.inputTag
	filters := buildFilters(tag)

	for {
		ev, ok := gtx.Event(filters...)
		if !ok {
			break
		}

		if len(win.tabs) == 0 {
			continue
		}
		active := win.tabs[win.activeTab]

		switch e := ev.(type) {

		case pointer.Event:
			if e.Kind == pointer.Scroll {
				if e.Scroll.Y > 0 {
					active.term.Scroll(-3)
				} else if e.Scroll.Y < 0 {
					active.term.Scroll(3)
				}
			}

		case key.FocusEvent:
			win.focused = e.Focus

		case key.Event:
			if e.State != key.Press {
				continue
			}

			isCtrl := e.Modifiers.Contain(key.ModCtrl)
			isShift := e.Modifiers.Contain(key.ModShift)

			// ── window-level binds ────────────────────────────────────────
			if isCtrl && isShift {
				switch e.Name {
				case "T":
					win.newTab()
					continue
				case "W":
					win.closeTab(win.activeTab)
					continue
				}
			}

			if isCtrl && !isShift {
				switch e.Name {
				case key.NamePageDown:
					win.activeTab = (win.activeTab + 1) % len(win.tabs)
					win.w.Invalidate()
					continue
				case key.NamePageUp:
					win.activeTab = (win.activeTab - 1 + len(win.tabs)) % len(win.tabs)
					win.w.Invalidate()
					continue
				}
			}

			// ── scroll binds (Shift + arrow / PgUp / PgDn) ───────────────
			if e.Modifiers == key.ModShift {
				switch e.Name {
				case key.NameUpArrow:
					active.term.Scroll(1)
					continue
				case key.NameDownArrow:
					active.term.Scroll(-1)
					continue
				case key.NamePageUp:
					active.term.Scroll(10)
					continue
				case key.NamePageDown:
					active.term.Scroll(-10)
					continue
				}
			}

			// ── forward everything else to the PTY ────────────────────────
			b := terminal.KeyToBytes(e, active.term.AppCursorKeys())
			if len(b) > 0 {
				active.term.Scroll(-999999)
				active.pty.Write(b) //nolint:errcheck
			}

		case key.EditEvent:
			// Space is already handled via key.NameSpace in KeyToBytes.
			if active.pty != nil && e.Text != " " {
				active.term.Scroll(-999999)
				active.pty.Write([]byte(e.Text)) //nolint:errcheck
			}
		}
	}
}

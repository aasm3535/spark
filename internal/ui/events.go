package ui

import (
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"yutug.lol/spark/internal/config"
	"yutug.lol/spark/internal/terminal"
)

// buildFilters returns the full list of input event filters for the terminal.
// Binding-specific filters are merged in from the BindingManager.
func buildFilters(tag *struct{}, bm *config.BindingManager) []event.Filter {
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
		key.Filter{Focus: tag, Name: `\`, Required: ctrl},
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

	// Ctrl+A–Z; T and W also accept Shift (covered by binding manager too,
	// but we keep them here so raw PTY passthrough still works).
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

	// Merge in any extra filters required by the binding manager.
	for _, bf := range bm.Filters(tag) {
		f = append(f, bf)
	}

	return f
}

// handleEvents processes all pending input events for the current frame.
func (win *Window) handleEvents(gtx layout.Context) {
	tag := &win.inputTag
	filters := buildFilters(tag, win.bindings)

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

			// ── Check binding manager first ───────────────────────────────
			if action := win.bindings.Resolve(e); action != config.ActionNone {
				win.handleAction(action)
				continue
			}

			// ── Forward everything else to the PTY ────────────────────────
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

// handleAction executes a resolved Action against the current window state.
func (win *Window) handleAction(action config.Action) {
	switch action {
	case config.ActionNewTab:
		win.newTab() //nolint:errcheck

	case config.ActionCloseTab:
		win.closeTab(win.activeTab)

	case config.ActionNextTab:
		if len(win.tabs) > 0 {
			win.activeTab = (win.activeTab + 1) % len(win.tabs)
			win.w.Invalidate()
		}

	case config.ActionPrevTab:
		if len(win.tabs) > 0 {
			win.activeTab = (win.activeTab - 1 + len(win.tabs)) % len(win.tabs)
			win.w.Invalidate()
		}

	case config.ActionScrollUp:
		if active := win.active(); active != nil {
			active.term.Scroll(1)
		}

	case config.ActionScrollDown:
		if active := win.active(); active != nil {
			active.term.Scroll(-1)
		}

	case config.ActionScrollPageUp:
		if active := win.active(); active != nil {
			active.term.Scroll(10)
		}

	case config.ActionScrollPageDown:
		if active := win.active(); active != nil {
			active.term.Scroll(-10)
		}
	}
}

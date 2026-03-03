package config

import "gioui.org/io/key"

// Action is a logical terminal action that can be bound to a key chord.
type Action int

const (
	ActionNone Action = iota

	// Tab management
	ActionNewTab
	ActionCloseTab
	ActionNextTab
	ActionPrevTab

	// Scrollback
	ActionScrollUp
	ActionScrollDown
	ActionScrollPageUp
	ActionScrollPageDown
)

// String returns a human-readable name for the action.
func (a Action) String() string {
	switch a {
	case ActionNewTab:
		return "new_tab"
	case ActionCloseTab:
		return "close_tab"
	case ActionNextTab:
		return "next_tab"
	case ActionPrevTab:
		return "prev_tab"
	case ActionScrollUp:
		return "scroll_up"
	case ActionScrollDown:
		return "scroll_down"
	case ActionScrollPageUp:
		return "scroll_page_up"
	case ActionScrollPageDown:
		return "scroll_page_down"
	default:
		return "none"
	}
}

// binding pairs a parsed Chord with the Action it triggers.
type binding struct {
	chord  Chord
	action Action
}

// BindingManager resolves key events into Actions based on the active keybind
// configuration. Build it once from a Config and reuse across frames.
type BindingManager struct {
	bindings []binding
}

// NewBindingManager creates a BindingManager from the resolved keybinds in cfg.
func NewBindingManager(cfg *Config) *BindingManager {
	kb := cfg.ResolvedKeybinds()

	pairs := []struct {
		chord  string
		action Action
	}{
		{kb.NewTab, ActionNewTab},
		{kb.CloseTab, ActionCloseTab},
		{kb.NextTab, ActionNextTab},
		{kb.PrevTab, ActionPrevTab},
		{kb.ScrollUp, ActionScrollUp},
		{kb.ScrollDown, ActionScrollDown},
		{kb.ScrollPageUp, ActionScrollPageUp},
		{kb.ScrollPageDown, ActionScrollPageDown},
	}

	bm := &BindingManager{}
	for _, p := range pairs {
		if c, ok := ParseChord(p.chord); ok {
			bm.bindings = append(bm.bindings, binding{chord: c, action: p.action})
		}
	}
	return bm
}

// Resolve returns the Action bound to the given key event, or ActionNone if
// no binding matches. Only key.Press events can match.
func (bm *BindingManager) Resolve(e key.Event) Action {
	if e.State != key.Press {
		return ActionNone
	}
	for _, b := range bm.bindings {
		if b.chord.Matches(e) {
			return b.action
		}
	}
	return ActionNone
}

// Chords returns all key.Filter entries needed to receive the bound key events.
// Pass the returned slice (along with all other filters) to gtx.Event.
func (bm *BindingManager) Filters(tag *struct{}) []key.Filter {
	seen := make(map[Chord]bool)
	var filters []key.Filter

	for _, b := range bm.bindings {
		if seen[b.chord] {
			continue
		}
		seen[b.chord] = true
		filters = append(filters, key.Filter{
			Focus:    tag,
			Name:     b.chord.Name,
			Required: b.chord.Mods,
		})
	}
	return filters
}

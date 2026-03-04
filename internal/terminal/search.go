package terminal

import (
	"strings"
	"unicode/utf8"
)

// Match represents a found text in the buffer.
type Match struct {
	Row int // 0-based index from top of scrollback to bottom of screen
	Col int // starting column
	Len int // length of match in runes/cells
}

// FindAll searches the entire scrollback and screen for the given query.
func (t *Terminal) FindAll(query string) []Match {
	t.mu.Lock()
	defer t.mu.Unlock()

	if query == "" {
		return nil
	}

	query = strings.ToLower(query)
	var matches []Match

	// Combine scrollback and screen sizes
	totalRows := len(t.scrollback) + len(t.screen)

	for row := 0; row < totalRows; row++ {
		var line []Cell
		if row < len(t.scrollback) {
			line = t.scrollback[row]
		} else {
			line = t.screen[row-len(t.scrollback)]
		}

		// Convert line to string
		var sb strings.Builder
		for _, c := range line {
			if c.Ch == 0 {
				sb.WriteRune(' ')
			} else {
				sb.WriteRune(c.Ch)
			}
		}

		str := strings.ToLower(sb.String())

		// Find all occurrences in this line
		offset := 0
		for {
			idx := strings.Index(str[offset:], query)
			if idx == -1 {
				break
			}

			// strings.Index returns byte index, we need rune index (cell index).
			// We can count runes up to offset+idx.
			runeIdx := utf8.RuneCountInString(str[:offset+idx])
			queryLen := utf8.RuneCountInString(query)

			matches = append(matches, Match{
				Row: row,
				Col: runeIdx,
				Len: queryLen,
			})

			// advance byte offset
			// advance by at least 1 to avoid infinite loop if query is empty (though handled above)
			step := len(query)
			if step == 0 {
				step = 1
			}
			offset += idx + step
		}
	}

	return matches
}

// SearchNext scrolls to the next search match.
func (t *Terminal) SearchNext() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.searchResults) == 0 {
		return
	}

	// Calculate visible viewport range
	sbLen := len(t.scrollback)
	viewportTop := sbLen - t.scrollOffset
	viewportBottom := viewportTop + t.rows - 1

	// Find the first match that is below the current viewport
	for _, m := range t.searchResults {
		if m.Row > viewportBottom {
			// Scroll so this match is at the top of the viewport
			t.scrollOffset = sbLen - m.Row
			return
		}
	}

	// Wrap around to first match
	t.scrollOffset = sbLen - t.searchResults[0].Row
}

// SearchPrev scrolls to the previous search match.
func (t *Terminal) SearchPrev() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.searchResults) == 0 {
		return
	}

	// Calculate visible viewport range
	sbLen := len(t.scrollback)
	viewportTop := sbLen - t.scrollOffset

	// Iterate backwards to find a match above the viewport
	for i := len(t.searchResults) - 1; i >= 0; i-- {
		m := t.searchResults[i]
		if m.Row < viewportTop {
			// Scroll so this match is at the top of the viewport
			t.scrollOffset = sbLen - m.Row
			return
		}
	}

	// Wrap around to last match
	lastMatch := t.searchResults[len(t.searchResults)-1]
	t.scrollOffset = sbLen - lastMatch.Row
}

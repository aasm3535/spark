package terminal

import (
	"image/color"
	"strings"
	"sync"
	"unicode/utf8"
)

// ─── Constants ───────────────────────────────────────────────────────────────

const (
	DefaultCols   = 220
	DefaultRows   = 50
	MaxScrollback = 4000
)

// ─── Terminal ─────────────────────────────────────────────────────────────────

// Invalidator is implemented by app.Window – called after every write so the
// UI schedules a redraw.
type Invalidator interface {
	Invalidate()
}

// Terminal holds the full VT/ANSI state: screen buffer, cursor, SGR attrs,
// alternate screen and scrollback.
type Terminal struct {
	mu sync.Mutex

	cols, rows   int
	screen       [][]Cell
	scrollback   [][]Cell
	scrollOffset int

	curX, curY     int
	savedX, savedY int

	fgColor color.NRGBA
	bgColor color.NRGBA
	bold    bool

	// escape-sequence parser
	inEscape bool
	escBuf   strings.Builder

	// alternate screen
	altScreen        [][]Cell
	altCurX, altCurY int
	inAlt            bool

	// Search
	searchQuery   string
	searchResults []Match

	appCursorKeys bool

	marginTop, marginBottom int

	showCursor bool
	lineWrap   bool

	inv Invalidator
}

// New creates a Terminal with the given dimensions.
func New(cols, rows int, inv Invalidator) *Terminal {
	t := &Terminal{
		cols:         cols,
		rows:         rows,
		fgColor:      ColorText,
		bgColor:      ColorBg,
		marginTop:    0,
		marginBottom: rows - 1,
		showCursor:   true,
		lineWrap:     true,
		inv:          inv,
	}
	t.screen = t.makeScreen(cols, rows)
	return t
}

// ─── Screen helpers ───────────────────────────────────────────────────────────

func (t *Terminal) makeScreen(cols, rows int) [][]Cell {
	s := make([][]Cell, rows)
	for i := range s {
		s[i] = t.makeLine(cols)
	}
	return s
}

func (t *Terminal) makeLine(cols int) []Cell {
	line := make([]Cell, cols)
	for i := range line {
		line[i] = DefaultCell()
	}
	return line
}

// ─── Snapshot (for renderer) ──────────────────────────────────────────────────

// SearchHighlight defines a region in the terminal to highlight.
type SearchHighlight struct {
	StartCol, EndCol int
}

type Snapshot struct {
	Cols, Rows    int
	CurX, CurY    int
	ShowCursor    bool
	Screen        [][]Cell
	ScrollOffset  int
	ScrollTotal   int
	SearchMatches map[int][]SearchHighlight // row index -> highlights
}

// Snapshot returns a deep copy of the current screen state, safe to read
// from the render goroutine without holding the lock.
func (t *Terminal) Snapshot() Snapshot {
	t.mu.Lock()
	defer t.mu.Unlock()

	snap := Snapshot{
		Cols:          t.cols,
		Rows:          t.rows,
		CurX:          t.curX,
		CurY:          t.curY,
		ShowCursor:    t.showCursor,
		ScrollOffset:  t.scrollOffset,
		ScrollTotal:   len(t.scrollback),
		SearchMatches: make(map[int][]SearchHighlight),
	}

	if t.scrollOffset > 0 {
		snap.CurY += t.scrollOffset
	}

	snap.Screen = make([][]Cell, t.rows)
	sbLen := len(t.scrollback)

	for i := 0; i < t.rows; i++ {
		cp := make([]Cell, t.cols)
		vIdx := sbLen - t.scrollOffset + i

		if vIdx >= 0 && vIdx < sbLen {
			copy(cp, t.scrollback[vIdx])
		} else if vIdx >= sbLen {
			screenIdx := vIdx - sbLen
			if screenIdx < len(t.screen) {
				copy(cp, t.screen[screenIdx])
			}
		}
		snap.Screen[i] = cp

		// Map global match rows to visible snapshot rows
		if len(t.searchResults) > 0 {
			for _, m := range t.searchResults {
				if m.Row == vIdx {
					snap.SearchMatches[i] = append(snap.SearchMatches[i], SearchHighlight{
						StartCol: m.Col,
						EndCol:   m.Col + m.Len,
					})
				}
			}
		}
	}
	return snap
}

// SetSearch sets the active search query and computes results.
func (t *Terminal) SetSearch(query string) {
	// Must not hold t.mu when calling FindAll since FindAll takes the lock.
	// But actually, we need to restructure to avoid deadlock.
	matches := t.FindAll(query)

	t.mu.Lock()
	defer t.mu.Unlock()
	t.searchQuery = query
	t.searchResults = matches
}

// Scroll moves the viewport up (positive) or down (negative).
func (t *Terminal) Scroll(delta int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.scrollOffset += delta
	if t.scrollOffset > len(t.scrollback) {
		t.scrollOffset = len(t.scrollback)
	}
	if t.scrollOffset < 0 {
		t.scrollOffset = 0
	}

	if t.inv != nil {
		t.inv.Invalidate()
	}
}

// ─── io.Writer ────────────────────────────────────────────────────────────────

// Write implements io.Writer; called from the ConPTY reader goroutine.
func (t *Terminal) Write(p []byte) (int, error) {
	n := len(p)
	t.mu.Lock()
	for len(p) > 0 {
		if t.inEscape {
			t.escBuf.WriteByte(p[0])
			p = p[1:]
			if t.escBuf.Len() > 1 {
				t.tryFinishEscape()
			}
			continue
		}

		r, size := utf8.DecodeRune(p)
		p = p[size:]

		switch r {
		case 0x1b:
			t.inEscape = true
			t.escBuf.Reset()
			t.escBuf.WriteRune(r)
		case '\r':
			t.curX = 0
		case '\n':
			t.newline()
		case '\t':
			t.curX = ((t.curX / 8) + 1) * 8
			if t.curX >= t.cols {
				t.curX = t.cols - 1
			}
		case '\b':
			if t.curX > 0 {
				t.curX--
			}
		case 0x07: // BEL – ignore
		default:
			if r >= 0x20 {
				t.putRune(r)
			}
		}
	}
	t.mu.Unlock()

	if t.inv != nil {
		t.inv.Invalidate()
	}
	return n, nil
}

func (t *Terminal) putRune(r rune) {
	if t.curX >= t.cols {
		if t.lineWrap {
			t.curX = 0
			t.newline()
		} else {
			t.curX = t.cols - 1
		}
	}
	if t.curY >= 0 && t.curY < t.rows && t.curX >= 0 && t.curX < t.cols {
		t.screen[t.curY][t.curX] = Cell{
			Ch:   r,
			Fg:   t.fgColor,
			Bg:   t.bgColor,
			Bold: t.bold,
		}
	}
	t.curX++
}

func (t *Terminal) newline() {
	if t.curY == t.marginBottom {
		if t.marginTop == 0 && t.marginBottom == t.rows-1 {
			line := t.screen[0]
			t.scrollback = append(t.scrollback, line)
			if len(t.scrollback) > MaxScrollback {
				t.scrollback = t.scrollback[len(t.scrollback)-MaxScrollback:]
			}
			// Если пользователь скроллит вверх — держим viewport на месте
			if t.scrollOffset > 0 {
				t.scrollOffset++
				if t.scrollOffset > len(t.scrollback) {
					t.scrollOffset = len(t.scrollback)
				}
			}
		}
		t.screen = append(t.screen[:t.marginTop], t.screen[t.marginTop+1:]...)
		blank := t.makeLine(t.cols)
		if t.marginBottom == t.rows-1 {
			t.screen = append(t.screen, blank)
		} else {
			newScreen := make([][]Cell, 0, t.rows)
			newScreen = append(newScreen, t.screen[:t.marginBottom]...)
			newScreen = append(newScreen, blank)
			newScreen = append(newScreen, t.screen[t.marginBottom:]...)
			t.screen = newScreen
		}
	} else if t.curY < t.rows-1 {
		t.curY++
	}
}

// ─── Resize ───────────────────────────────────────────────────────────────────

// Resize adapts the buffer to new dimensions, preserving existing content.
func (t *Terminal) AppCursorKeys() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.appCursorKeys
}

func (t *Terminal) Resize(cols, rows int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if cols == t.cols && rows == t.rows {
		return
	}
	newScreen := t.makeScreen(cols, rows)
	for y := 0; y < rows && y < t.rows; y++ {
		for x := 0; x < cols && x < t.cols; x++ {
			newScreen[y][x] = t.screen[y][x]
		}
	}
	t.screen = newScreen
	t.cols = cols
	t.rows = rows
	t.marginTop = 0
	t.marginBottom = rows - 1
	if t.curX >= cols {
		t.curX = cols - 1
	}
	if t.curY >= rows {
		t.curY = rows - 1
	}
}

// ─── Escape sequence parser ───────────────────────────────────────────────────

func (t *Terminal) tryFinishEscape() {
	s := t.escBuf.String()

	// OSC: ESC ] … BEL  or  ESC ] … ESC \
	if len(s) > 1 && s[1] == ']' {
		if strings.HasSuffix(s, "\x07") || strings.HasSuffix(s, "\x1b\\") {
			t.inEscape = false
			t.escBuf.Reset()
		}
		if t.escBuf.Len() > 512 {
			t.inEscape = false
			t.escBuf.Reset()
		}
		return
	}

	// CSI: ESC [ <params> <final>
	if len(s) > 1 && s[1] == '[' {
		body := s[2:]
		if len(body) == 0 {
			return
		}
		last := body[len(body)-1]
		if last >= 0x40 && last <= 0x7e {
			t.handleCSI(body[:len(body)-1], last)
			t.inEscape = false
			t.escBuf.Reset()
		}
		if t.escBuf.Len() > 256 {
			t.inEscape = false
			t.escBuf.Reset()
		}
		return
	}

	// Two-byte ESC sequences
	if len(s) == 2 {
		switch s[1] {
		case 'c':
			t.fullReset()
		case '7':
			t.savedX, t.savedY = t.curX, t.curY
		case '8':
			t.curX, t.curY = t.savedX, t.savedY
		case 'D':
			t.newline()
		case 'E':
			t.newline()
			t.curX = 0
		case 'M':
			if t.curY == t.marginTop {
				blank := t.makeLine(t.cols)
				t.screen = append(t.screen[:t.marginBottom], t.screen[t.marginBottom+1:]...)
				newScreen := make([][]Cell, 0, t.rows)
				newScreen = append(newScreen, t.screen[:t.marginTop]...)
				newScreen = append(newScreen, blank)
				newScreen = append(newScreen, t.screen[t.marginTop:]...)
				t.screen = newScreen
			} else if t.curY > 0 {
				t.curY--
			}
		}
		if s[1] >= 0x40 {
			t.inEscape = false
			t.escBuf.Reset()
		}
		return
	}

	if len(s) == 3 && (s[1] == '(' || s[1] == ')' || s[1] == '*' || s[1] == '+') {
		t.inEscape = false
		t.escBuf.Reset()
		return
	}

	if t.escBuf.Len() > 256 {
		t.inEscape = false
		t.escBuf.Reset()
	}
}

// ─── CSI dispatch ─────────────────────────────────────────────────────────────

func (t *Terminal) handleCSI(params string, final byte) {
	parts := strings.Split(params, ";")

	clamp := func(v, lo, hi int) int {
		if v < lo {
			return lo
		}
		if v > hi {
			return hi
		}
		return v
	}

	switch final {
	case '@':
		n := atoi(parts[0], 1)
		if n < 1 {
			n = 1
		}
		line := t.screen[t.curY]
		if t.curX+n < t.cols {
			copy(line[t.curX+n:], line[t.curX:])
		}
		for x := t.curX; x < t.curX+n && x < t.cols; x++ {
			line[x] = DefaultCell()
		}
	case 'A':
		n := atoi(parts[0], 1)
		t.curY = clamp(t.curY-n, 0, t.rows-1)
	case 'B':
		n := atoi(parts[0], 1)
		t.curY = clamp(t.curY+n, 0, t.rows-1)
	case 'C':
		n := atoi(parts[0], 1)
		t.curX = clamp(t.curX+n, 0, t.cols-1)
	case 'D':
		n := atoi(parts[0], 1)
		t.curX = clamp(t.curX-n, 0, t.cols-1)
	case 'G':
		n := atoi(parts[0], 1)
		t.curX = clamp(n-1, 0, t.cols-1)
	case 'd':
		n := atoi(parts[0], 1)
		t.curY = clamp(n-1, 0, t.rows-1)
	case 'H', 'f':
		row := atoi(parts[0], 1)
		col := 1
		if len(parts) >= 2 {
			col = atoi(parts[1], 1)
		}
		t.curY = clamp(row-1, 0, t.rows-1)
		t.curX = clamp(col-1, 0, t.cols-1)
	case 'J':
		switch atoi(parts[0], 0) {
		case 0:
			for x := t.curX; x < t.cols; x++ {
				t.screen[t.curY][x] = DefaultCell()
			}
			for y := t.curY + 1; y < t.rows; y++ {
				t.screen[y] = t.makeLine(t.cols)
			}
		case 1:
			for y := 0; y < t.curY; y++ {
				t.screen[y] = t.makeLine(t.cols)
			}
			for x := 0; x <= t.curX; x++ {
				t.screen[t.curY][x] = DefaultCell()
			}
		case 2, 3:
			for y := range t.screen {
				t.screen[y] = t.makeLine(t.cols)
			}
		}
	case 'K':
		switch atoi(parts[0], 0) {
		case 0:
			for x := t.curX; x < t.cols; x++ {
				t.screen[t.curY][x] = DefaultCell()
			}
		case 1:
			for x := 0; x <= t.curX; x++ {
				t.screen[t.curY][x] = DefaultCell()
			}
		case 2:
			t.screen[t.curY] = t.makeLine(t.cols)
		}
	case 'L':
		n := atoi(parts[0], 1)
		if t.curY >= t.marginTop && t.curY <= t.marginBottom {
			for i := 0; i < n; i++ {
				t.screen = append(t.screen[:t.marginBottom], t.screen[t.marginBottom+1:]...)
				newScreen := make([][]Cell, 0, t.rows)
				newScreen = append(newScreen, t.screen[:t.curY]...)
				newScreen = append(newScreen, t.makeLine(t.cols))
				newScreen = append(newScreen, t.screen[t.curY:]...)
				t.screen = newScreen
			}
		}
	case 'M':
		n := atoi(parts[0], 1)
		if t.curY >= t.marginTop && t.curY <= t.marginBottom {
			for i := 0; i < n; i++ {
				t.screen = append(t.screen[:t.curY], t.screen[t.curY+1:]...)
				newScreen := make([][]Cell, 0, t.rows)
				newScreen = append(newScreen, t.screen[:t.marginBottom]...)
				newScreen = append(newScreen, t.makeLine(t.cols))
				newScreen = append(newScreen, t.screen[t.marginBottom:]...)
				t.screen = newScreen
			}
		}
	case 'X':
		n := atoi(parts[0], 1)
		if n < 1 {
			n = 1
		}
		line := t.screen[t.curY]
		for x := t.curX; x < t.curX+n && x < t.cols; x++ {
			line[x] = DefaultCell()
		}
	case 'P':
		n := atoi(parts[0], 1)
		if n < 1 {
			n = 1
		}
		line := t.screen[t.curY]
		if t.curX+n < t.cols {
			copy(line[t.curX:], line[t.curX+n:])
		}
		for x := t.cols - n; x < t.cols; x++ {
			if x >= 0 {
				line[x] = DefaultCell()
			}
		}
	case 'S':
		n := atoi(parts[0], 1)
		for i := 0; i < n; i++ {
			if t.marginTop == 0 && t.marginBottom == t.rows-1 {
				t.scrollback = append(t.scrollback, t.screen[0])
			}
			t.screen = append(t.screen[:t.marginTop], t.screen[t.marginTop+1:]...)
			blank := t.makeLine(t.cols)
			if t.marginBottom == t.rows-1 {
				t.screen = append(t.screen, blank)
			} else {
				newScreen := make([][]Cell, 0, t.rows)
				newScreen = append(newScreen, t.screen[:t.marginBottom]...)
				newScreen = append(newScreen, blank)
				newScreen = append(newScreen, t.screen[t.marginBottom:]...)
				t.screen = newScreen
			}
		}
	case 'T':
		n := atoi(parts[0], 1)
		for i := 0; i < n; i++ {
			t.screen = append(t.screen[:t.marginBottom], t.screen[t.marginBottom+1:]...)
			blank := t.makeLine(t.cols)
			newScreen := make([][]Cell, 0, t.rows)
			newScreen = append(newScreen, t.screen[:t.marginTop]...)
			newScreen = append(newScreen, blank)
			newScreen = append(newScreen, t.screen[t.marginTop:]...)
			t.screen = newScreen
		}
	case 'm':
		t.handleSGR(parts)
	case 'h', 'l':
		for _, p := range parts {
			p = strings.TrimPrefix(p, "?")
			switch p {
			case "1049", "47":
				if final == 'h' {
					t.enterAlt()
				} else {
					t.exitAlt()
				}
			case "1":
				t.appCursorKeys = final == 'h'
			case "25":
				t.showCursor = final == 'h'
			case "7":
				t.lineWrap = final == 'h'
			}
		}
	case 'r':
		top := 1
		bottom := t.rows
		if len(parts) >= 1 && parts[0] != "" {
			top = atoi(parts[0], 1)
		}
		if len(parts) >= 2 && parts[1] != "" {
			bottom = atoi(parts[1], t.rows)
		}
		t.marginTop = clamp(top-1, 0, t.rows-1)
		t.marginBottom = clamp(bottom-1, 0, t.rows-1)
		if t.marginTop > t.marginBottom {
			t.marginTop, t.marginBottom = 0, t.rows-1
		}
		t.curX, t.curY = 0, t.marginTop
	case 's':
		t.savedX, t.savedY = t.curX, t.curY
	case 'u':
		t.curX, t.curY = t.savedX, t.savedY
	case 'n': // device status – ignore
	}
}

// ─── SGR (colours + attributes) ───────────────────────────────────────────────

func (t *Terminal) handleSGR(parts []string) {
	i := 0
	for i < len(parts) {
		n := atoi(parts[i], 0)
		switch {
		case n == 0:
			t.fgColor = ColorText
			t.bgColor = ColorBg
			t.bold = false
		case n == 1:
			t.bold = true
		case n == 22:
			t.bold = false
		case n >= 30 && n <= 37:
			idx := n - 30
			if t.bold {
				idx += 8
			}
			t.fgColor = AnsiColors[idx]
		case n == 38:
			if i+2 < len(parts) && atoi(parts[i+1], -1) == 5 {
				t.fgColor = Ansi256(atoi(parts[i+2], 0))
				i += 2
			} else if i+4 < len(parts) && atoi(parts[i+1], -1) == 2 {
				t.fgColor = rgb(parts[i+2], parts[i+3], parts[i+4])
				i += 4
			}
		case n == 39:
			t.fgColor = ColorText
		case n >= 40 && n <= 47:
			t.bgColor = AnsiColors[n-40]
		case n == 48:
			if i+2 < len(parts) && atoi(parts[i+1], -1) == 5 {
				t.bgColor = Ansi256(atoi(parts[i+2], 0))
				i += 2
			} else if i+4 < len(parts) && atoi(parts[i+1], -1) == 2 {
				t.bgColor = rgb(parts[i+2], parts[i+3], parts[i+4])
				i += 4
			}
		case n == 49:
			t.bgColor = ColorBg
		case n >= 90 && n <= 97:
			t.fgColor = AnsiColors[n-90+8]
		case n >= 100 && n <= 107:
			t.bgColor = AnsiColors[n-100+8]
		}
		i++
	}
}

// ─── Alternate screen ─────────────────────────────────────────────────────────

func (t *Terminal) enterAlt() {
	if t.inAlt {
		return
	}
	t.altScreen = t.screen
	t.altCurX, t.altCurY = t.curX, t.curY
	t.screen = t.makeScreen(t.cols, t.rows)
	t.curX, t.curY = 0, 0
	t.inAlt = true
}

func (t *Terminal) exitAlt() {
	if !t.inAlt {
		return
	}
	t.screen = t.altScreen
	t.curX, t.curY = t.altCurX, t.altCurY
	t.altScreen = nil
	t.inAlt = false
}

func (t *Terminal) fullReset() {
	t.fgColor = ColorText
	t.bgColor = ColorBg
	t.bold = false
	t.screen = t.makeScreen(t.cols, t.rows)
	t.marginTop, t.marginBottom = 0, t.rows-1
	t.curX, t.curY = 0, 0
	t.scrollback = nil
	t.inAlt = false
	t.altScreen = nil
}

// ─── Small helpers ────────────────────────────────────────────────────────────

func atoi(s string, def int) int {
	if s == "" {
		return def
	}
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return def
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func rgb(rs, gs, bs string) color.NRGBA {
	return color.NRGBA{
		R: uint8(atoi(rs, 0)),
		G: uint8(atoi(gs, 0)),
		B: uint8(atoi(bs, 0)),
		A: 255,
	}
}

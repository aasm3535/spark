# spark — contributor & agent guide

A reference for anyone (human or AI agent) working on this codebase.

---

## Project layout

```
main.go
internal/
  config/
    config.go               — Config struct, Load/Save, DefaultConfig
    keybinds_parser.go      — ParseChord, Chord.Matches
    keybinds_manager.go     — Action enum, BindingManager
  pty/                      — PTY interface + platform implementations
  terminal/                 — VT/ANSI buffer, escape parser, key mapping
  ui/
    window.go               — Window, New, Layout, ReadyForClose
    tab.go                  — Tab, newTab, closeTab, active, cleanup
    tabbar.go               — thin adapter: syncs Tab list → components.TabBar
    terminal_view.go        — layoutTerminal + scrollbar
    events.go               — buildFilters, handleEvents, handleAction
    components/
      theme.go              — palette vars, NewTheme, blendColor
      titlebar.go           — TitleBar struct + Layout
      tabbar.go             — TabBar struct + Layout → TabBarResult
      renderer.go           — Renderer struct + Layout, ColsRows
```

---

## How to write a component

Every reusable UI element lives in `internal/ui/components/` as its own file.

### Rules

1. **One struct per concept.**
   The struct holds only *widget state* — things that persist across frames
   (clickables, scrollbars, toggles). It must not hold layout parameters or
   theme colours; those are passed in at call-time.

2. **Layout method signature.**
   ```go
   func (c *MyComponent) Layout(gtx layout.Context, th *material.Theme, /* data */) layout.Dimensions
   ```
   If the component can produce events (e.g. a button was clicked), return a
   result struct instead of bare `layout.Dimensions`:
   ```go
   func (c *MyComponent) Layout(gtx layout.Context, th *material.Theme) (layout.Dimensions, MyComponentResult)
   ```

3. **Result structs, not callbacks.**
   Communicate events back to the caller via a plain result struct, not
   function callbacks or channels. The caller decides what to do with the
   event *after* Layout returns.
   ```go
   type MyComponentResult struct {
       Event     MyComponentEvent
       // any extra fields needed by the caller
   }
   ```

4. **Use palette colours, not hardcoded values.**
   All colours come from the vars in `components/theme.go`
   (`ColorBg`, `ColorText`, `ColorTitleBar`, etc.).
   Never write `color.NRGBA{R: 30, ...}` inline unless it is a *local*
   highlight derived at draw-time (e.g. hover highlight alpha).

5. **No business logic inside a component.**
   A component renders and reports interactions. It never calls `win.newTab()`
   or mutates global state. That work belongs in `ui/window.go` or
   `ui/events.go`.

6. **Keep Layout pure within a frame.**
   Do not spawn goroutines or mutate shared state inside Layout. The only
   side-effect allowed is calling `gtx.Execute` / `w.Invalidate` via a result
   that the *caller* acts on.

### Minimal example

```go
// components/statusbar.go
package components

import (
    "gioui.org/layout"
    "gioui.org/unit"
    "gioui.org/widget"
    "gioui.org/widget/material"
)

// StatusBar shows a single line of status text with a clickable area.
type StatusBar struct {
    Click widget.Clickable
}

type StatusBarResult struct {
    Clicked bool
}

func (s *StatusBar) Layout(
    gtx layout.Context,
    th *material.Theme,
    text string,
) (layout.Dimensions, StatusBarResult) {
    var result StatusBarResult

    if s.Click.Clicked(gtx) {
        result.Clicked = true
    }

    dims := s.Click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
        return layout.Inset{
            Top: unit.Dp(4), Bottom: unit.Dp(4),
            Left: unit.Dp(8), Right: unit.Dp(8),
        }.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
            lbl := material.Label(th, unit.Sp(11), text)
            lbl.Color = ColorTitleText
            return lbl.Layout(gtx)
        })
    })

    return dims, result
}
```

---

## How to add a keybind

### 1. Register the action

In `internal/config/keybinds_manager.go`, add a constant to the `Action` enum
and a case to `Action.String()`:

```go
const (
    ...
    ActionCopyText   // new
)

func (a Action) String() string {
    switch a {
    ...
    case ActionCopyText:
        return "copy_text"
    }
}
```

### 2. Add it to Keybinds

In `internal/config/config.go`, add a field to the `Keybinds` struct,
a default value in `DefaultKeybinds()`, and a merge case in `Keybinds.Merge()`:

```go
type Keybinds struct {
    ...
    CopyText string `json:"copy_text"`
}

func DefaultKeybinds() Keybinds {
    return Keybinds{
        ...
        CopyText: "Ctrl+Shift+C",
    }
}

func (d Keybinds) Merge(o Keybinds) Keybinds {
    ...
    if o.CopyText != "" { d.CopyText = o.CopyText }
    return d
}
```

### 3. Wire it in BindingManager

In `NewBindingManager`, add the pair to the `pairs` slice:

```go
{kb.CopyText, ActionCopyText},
```

### 4. Handle it in handleAction

In `internal/ui/events.go`, add a case to `handleAction`:

```go
case config.ActionCopyText:
    if active := win.active(); active != nil {
        // ... do the work
    }
```

That's it. The user can now remap the key in `~/.spark/config.json`:

```json
{
  "keybinds": {
    "copy_text": "Ctrl+C"
  }
}
```

---

## Config file

Location: `~/.spark/config.json`

Missing fields fall back to their defaults — users only need to specify what
they want to override.

### Full schema (with defaults)

```json
{
  "font_family": "Iosevka Fixed, Go Mono, monospace",
  "font_size": 14,
  "theme": "default",

  "custom_theme": {
    "bg":               "#121218",
    "fg":               "#dcdce6",
    "title_bar":        "#16161e",
    "title_text":       "#a0a0b4",
    "cursor":           "#82c8ff",
    "btn_hover_close":  "#c42b1c",
    "btn_hover_neutral":"#ffffff12",
    "tab_active_bg":    "#121218",
    "tab_inactive_bg":  "#16161e",
    "tab_hover_bg":     "#1e1e28"
  },

  "keybinds": {
    "new_tab":           "Ctrl+Shift+T",
    "close_tab":         "Ctrl+Shift+W",
    "next_tab":          "Ctrl+PageDown",
    "prev_tab":          "Ctrl+PageUp",
    "scroll_up":         "Shift+UpArrow",
    "scroll_down":       "Shift+DownArrow",
    "scroll_page_up":    "Shift+PageUp",
    "scroll_page_down":  "Shift+PageDown"
  }
}
```

### Chord format

A chord is a `+`-separated list of modifiers and a key name:

```
Ctrl+Shift+T
Shift+PageUp
Ctrl+PageDown
Alt+F4
```

Recognised modifiers: `Ctrl`, `Shift`, `Alt` (case-insensitive).

Recognised special key names (case-insensitive):

| Config name          | Key              |
|----------------------|------------------|
| `PageUp` / `PgUp`    | Page Up          |
| `PageDown` / `PgDn`  | Page Down        |
| `UpArrow` / `Up`     | ↑                |
| `DownArrow` / `Down` | ↓                |
| `LeftArrow` / `Left` | ←                |
| `RightArrow` / `Right`| →               |
| `Home`               | Home             |
| `End`                | End              |
| `Return` / `Enter`   | Enter            |
| `Backspace`          | Backspace        |
| `Delete`             | Delete           |
| `Tab`                | Tab              |
| `Space`              | Space            |
| `Escape` / `Esc`     | Escape           |
| `F1`–`F12`           | Function keys    |
| `A`–`Z`              | Letter keys      |

An empty string `""` disables the binding entirely.

---

## Coding conventions

- **No global mutable state outside `components/theme.go`.**
  Theme palette vars are the one accepted exception because Gio's widget
  library requires concrete colours at draw-time.

- **Errors from PTY writes are intentionally ignored** (`//nolint:errcheck`).
  The read goroutine in `tab.go` will surface the disconnect.

- **Lock scope in `terminal.go`.**
  The mutex in `Terminal` is held only for the minimum scope needed. Never
  call external code (e.g. `Invalidate`) while holding the lock.

- **No init() functions.**
  Initialise state explicitly in constructors (`New`, `NewTheme`, etc.).

- **File naming.**
  Each file has one clear responsibility. If a file starts needing two
  unrelated `import` groups, split it.
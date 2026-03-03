# spark

A minimal terminal emulator for Windows and Linux, built with [Gio](https://gioui.org).

## Features

- ConPTY backend on Windows (PowerShell / cmd.exe)
- Unix PTY backend on Linux and WSL ($SHELL / bash)
- ANSI / VT100 color and attribute support (16, 256, truecolor)
- Custom borderless window with native-style controls
- Embedded Iosevka Fixed font
- Full keyboard support: Ctrl+A–Z, F1–F12, arrows, etc.
- Multiple tabs (Ctrl+Shift+T / Ctrl+Shift+W)
- Configurable keybinds and theme via `~/.spark/config.json`

## Requirements

**Windows** — Windows 10 1809 or later, Go 1.22+

**Linux / WSL** — Go 1.22+, X11 or Wayland, and the following packages:

```
sudo apt install -y libx11-dev libxcursor-dev libxrandr-dev libxi-dev libgl1-mesa-dev
```

## Build

```
go build .
```

The binary is placed in the current directory. On Windows it will open without
a console window automatically via the manifest embedded at build time.

## Configuration

spark reads `~/.spark/config.json` on startup and creates it with defaults if
it does not exist. All fields are optional — only specify what you want to
change.

```json
{
  "font_family": "Iosevka Fixed, Go Mono, monospace",
  "font_size": 14,
  "theme": "default",

  "custom_theme": {
    "bg":                "#121218",
    "fg":                "#dcdce6",
    "title_bar":         "#16161e",
    "title_text":        "#a0a0b4",
    "cursor":            "#82c8ff",
    "btn_hover_close":   "#c42b1c",
    "btn_hover_neutral": "#ffffff12",
    "tab_active_bg":     "#121218",
    "tab_inactive_bg":   "#16161e",
    "tab_hover_bg":      "#1e1e28"
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

## Default keybinds

| Action           | Default          |
|------------------|------------------|
| New tab          | Ctrl+Shift+T     |
| Close tab        | Ctrl+Shift+W     |
| Next tab         | Ctrl+PageDown    |
| Previous tab     | Ctrl+PageUp      |
| Scroll up        | Shift+↑          |
| Scroll down      | Shift+↓          |
| Scroll page up   | Shift+PageUp     |
| Scroll page down | Shift+PageDown   |

## Project layout

```
main.go
internal/
  config/         — Config, keybind parser and binding manager
  pty/            — PTY interface + Windows (ConPTY) and Linux implementations
  terminal/       — VT/ANSI buffer, escape parser, key mapping
  ui/
    components/   — reusable Gio components (TitleBar, TabBar, Renderer, Theme)
    window.go     — root window and layout
    events.go     — input event handling
```

See [AGENTS.md](AGENTS.md) for contribution and architecture guidelines.

## License

MIT. Copyright 2026.
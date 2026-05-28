# spark

A minimal, hackable terminal emulator with a sidebar and AI awareness. Built with [Gio](https://gioui.org).

---

**Sidebar tabs** with live AI agent status, **speech-to-text** input, **search**, **command palette**, full **truecolor** support, and everything configurable through a single JSON file.

---

## Quick start

```
go build .
./spark
```

That's it. On first run spark creates `~/.spark/config.json` with sensible defaults.

## Keybinds

| Action | Key |
|---|---|
| New tab | Ctrl+Shift+T |
| Close tab | Ctrl+Shift+W |
| Next / Prev tab | Ctrl+PageDown / PageUp |
| Scroll | Shift+Up / Down |
| Scroll page | Shift+PageUp / PageDown |
| Copy / Paste | Ctrl+Shift+C / V |
| Find | Ctrl+Shift+F |
| Command palette | Ctrl+Shift+P |
| Voice input | Ctrl+Shift+H |
| Zoom in / out | Ctrl+Scroll |

All keybinds are remappable in config.

## Features

- Resizable sidebar with tab list (drag to resize, 120-400 dp)
- AI agent detection -- sidebar shows working/idle state for Claude Code, Opencode, Pi
- Speech-to-text via OpenAI Whisper, Nvidia NIM, or AssemblyAI
- VT100/ANSI truecolor (16, 256, 24-bit), box drawing, block elements, braille
- Embedded Geist Mono font + auto-detected Nerd Font fallback
- Text selection, copy/paste, bracketed paste
- Search with highlighting
- Command palette
- Custom borderless window with native controls
- ConPTY (Windows) / Unix PTY (Linux, WSL)
- Full keyboard: Ctrl+A-Z, F1-F12, arrows, etc.
- Themed via `~/.spark/config.json` -- colors, font, padding

## Config

`~/.spark/config.json` -- all fields optional, override only what you need:

```json
{
  "font_family": "Geist Mono",
  "font_size": 14,
  "padding": 5,
  "custom_theme": {
    "bg": "#121218",
    "fg": "#dcdce6",
    "title_bar": "#16161e",
    "cursor": "#82c8ff"
  },
  "stt": {
    "provider": "openai",
    "api_key": "",
    "model": "whisper-1"
  }
}
```

See [AGENTS.md](AGENTS.md) for the full schema and contribution guide.

## Requirements

- **Windows**: 10 1809+, Go 1.22+
- **Linux/WSL**: Go 1.22+, X11 or Wayland

```
sudo apt install -y libx11-dev libxcursor-dev libxrandr-dev libxi-dev libgl1-mesa-dev
```

## License

MIT

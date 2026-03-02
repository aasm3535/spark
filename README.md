# spark

A minimal terminal emulator for Windows and Linux, built with Gio.

## Features

- ConPTY backend on Windows (PowerShell / cmd.exe)
- Unix PTY backend on Linux and WSL ($SHELL / bash)
- ANSI / VT100 color support (16, 256, truecolor)
- Custom borderless window with native-style controls
- Embedded Iosevka Fixed font
- Full keyboard support: Ctrl+A–Z, F1–F12, arrows, etc.

## Requirements

Windows:
- Windows 10 1809 or later (ConPTY)
- Go 1.22 or later

Linux / WSL:
- Go 1.22 or later
- X11 or Wayland display server
- Build dependencies: libx11-dev libxcursor-dev libxrandr-dev libxi-dev libgl1-mesa-dev

## Build

Windows:

    go build -ldflags="-H windowsgui -s -w" -o bin/spark.exe .

Linux / WSL:

    sudo apt install -y libx11-dev libxcursor-dev libxrandr-dev libxi-dev libgl1-mesa-dev
    go build -ldflags="-s -w" -o bin/spark .

Note: cross-compilation from Windows to Linux is not supported because Gio
requires CGO on Linux. Build the Linux binary inside WSL or a native Linux
environment.

## Running in WSL

WSL2 with WSLg (Windows 11) supports GUI apps natively — just run:

    ./bin/spark

On older WSL without WSLg you need an X server on Windows (VcXsrv, Xming)
and set the DISPLAY variable:

    export DISPLAY=:0
    ./bin/spark

## Custom font

Drop any .ttf or .otf file into assets/fonts/ and rebuild.
The font will be registered as the primary terminal face.

## Project layout

    main.go
    bin/                  compiled binaries
    assets/fonts/         font files (embedded at build time)
    internal/
        assets/           go:embed declarations
        pty/              PTY interface + Windows (ConPTY) and Linux implementations
        terminal/         VT/ANSI buffer, parser, key mapping
        ui/               Gio window, titlebar, renderer, theme

## License

MIT. Copyright 2026.
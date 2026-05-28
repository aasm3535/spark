package ui

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/transfer"
	"gioui.org/layout"
	"yutug.lol/spark/internal/config"
	"yutug.lol/spark/internal/stt"
	"yutug.lol/spark/internal/terminal"
)

// sttResult is sent from the transcription goroutine to the main thread
// so that all state mutations happen on a single goroutine.
type sttResult struct {
	text string
	err  error
}

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
		pointer.Filter{
			Target:  tag,
			Kinds:   pointer.Press | pointer.Release | pointer.Move | pointer.Drag,
		},
		key.FocusFilter{Target: tag},
		transfer.TargetFilter{Target: tag, Type: "text/plain"},
		transfer.TargetFilter{Target: tag, Type: "application/text"},
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
				if e.Modifiers.Contain(key.ModCtrl) {
					if e.Scroll.Y > 0 {
						win.changeFontSize(-1)
					} else if e.Scroll.Y < 0 {
						win.changeFontSize(1)
					}
				} else if active.term.MouseTracking() > 0 {
					btn := 4
					if e.Scroll.Y > 0 {
						btn = 5
					}
					col := int(e.Position.X) / win.renderer.CellWidth
					row := int(e.Position.Y) / win.renderer.CellHeight
					active.term.HandleMouse(btn, col, row, true, false, false)
				} else {
					if e.Scroll.Y > 0 {
						active.term.Scroll(-3)
					} else if e.Scroll.Y < 0 {
						active.term.Scroll(3)
					}
				}
			} else if e.Kind == pointer.Press || e.Kind == pointer.Release || e.Kind == pointer.Move || e.Kind == pointer.Drag {
				active.MouseX = int(e.Position.X)
				if active.term.MouseTracking() > 0 {
					col := int(e.Position.X) / win.renderer.CellWidth
					row := int(e.Position.Y) / win.renderer.CellHeight
					btn := 0
					if e.Buttons&pointer.ButtonPrimary != 0 {
						btn = 1
					} else if e.Buttons&pointer.ButtonSecondary != 0 {
						btn = 3
					} else if e.Buttons&pointer.ButtonTertiary != 0 {
						btn = 2
					}
					press := e.Kind == pointer.Press
					release := e.Kind == pointer.Release
					motion := e.Kind == pointer.Move || e.Kind == pointer.Drag
					active.term.HandleMouse(btn, col, row, press, release, motion)
				} else {
					col := int(e.Position.X) / win.renderer.CellWidth
					row := int(e.Position.Y) / win.renderer.CellHeight

					snap := active.term.Snapshot()
					absRow := snap.ScrollTotal - snap.ScrollOffset + row

					switch e.Kind {
					case pointer.Press:
						if e.Buttons&pointer.ButtonPrimary != 0 {
							active.term.SelectionStart(absRow, col)
							win.w.Invalidate()
						}
					case pointer.Drag:
						active.term.SelectionUpdate(absRow, col)
						win.w.Invalidate()
					case pointer.Release:
						active.term.SelectionEnd()
					}
				}
			}

		case key.FocusEvent:
			win.focused = e.Focus
			active.term.HandleFocus(e.Focus)

		case key.Event:
			if e.State != key.Press {
				continue
			}

			if action := win.bindings.Resolve(e); action != config.ActionNone {
				win.handleAction(gtx, action)
				continue
			}

			// Fallback: support standard Ctrl+V for paste if not resolved
			if e.Modifiers == key.ModCtrl && e.Name == "V" {
				win.handleAction(gtx, config.ActionPaste)
				continue
			}

			if win.searchActive {
				continue
			}

			b := terminal.KeyToBytes(e, active.term.AppCursorKeys())
			if len(b) > 0 {
				active.term.Scroll(-999999)
				active.pty.Write(b) //nolint:errcheck
			}

		case key.EditEvent:
			if win.searchActive {
				continue
			}
			if active.pty != nil && e.Text != " " {
				active.term.Scroll(-999999)
				if active.term.BracketedPaste() && len(e.Text) > 1 {
					active.pty.Write([]byte("\x1b[200~")) //nolint:errcheck
					active.pty.Write([]byte(e.Text)) //nolint:errcheck
					active.pty.Write([]byte("\x1b[201~")) //nolint:errcheck
				} else {
					active.pty.Write([]byte(e.Text)) //nolint:errcheck
				}
			}

		case transfer.DataEvent:
			if active.pty != nil {
				rc := e.Open()
				data, err := io.ReadAll(rc)
				rc.Close() //nolint:errcheck
				if err == nil && len(data) > 0 {
					// Normalize line endings to \r for the PTY shell input
					text := string(data)
					text = strings.ReplaceAll(text, "\r\n", "\r")
					text = strings.ReplaceAll(text, "\n", "\r")
					data = []byte(text)

					active.term.Scroll(-999999)
					if active.term.BracketedPaste() {
						active.pty.Write([]byte("\x1b[200~")) //nolint:errcheck
						active.pty.Write(data) //nolint:errcheck
						active.pty.Write([]byte("\x1b[201~")) //nolint:errcheck
					} else {
						active.pty.Write(data) //nolint:errcheck
					}
				}
			}
		}
	}
}

// processSTTResults drains the STT result channel on the main thread.
// All state mutations (sttTranscribing, sttCancel, pty writes, toast) happen
// here so that the transcription goroutine never touches shared state directly.
func (win *Window) processSTTResults() {
	for {
		select {
		case result := <-win.sttResultCh:
			win.sttTranscribing = false
			win.sttCancel = nil

			if result.err != nil {
				win.ShowToast(fmt.Sprintf("STT Error: %v", result.err))
			} else {
				text := strings.TrimSpace(result.text)
				if text != "" {
					if active := win.active(); active != nil {
						active.pty.Write([]byte(text)) //nolint:errcheck
					}
				}
			}
			win.w.Invalidate()
		default:
			return
		}
	}
}

// handleAction executes a resolved Action against the current window state.
func (win *Window) handleAction(gtx layout.Context, action config.Action) {
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

	case config.ActionCommandPalette:
		win.cmdActive = !win.cmdActive
		if win.cmdActive {
			win.cmdPal.Editor.SetText("")
		}
		win.w.Invalidate()

	case config.ActionFind:
		win.searchActive = true
		win.w.Invalidate()

	case config.ActionCopyText:
		if active := win.active(); active != nil {
			text := active.term.SelectedText()
			if text != "" {
				gtx.Execute(clipboard.WriteCmd{
					Type: "application/text",
					Data: io.NopCloser(bytes.NewReader([]byte(text))),
				})
				win.ShowToast("Copied to clipboard")
			}
		}

	case config.ActionPaste:
		gtx.Execute(clipboard.ReadCmd{Tag: &win.inputTag})

	case config.ActionSTT:
		active := win.active()
		if active == nil {
			break
		}

		if win.sttRecording {
			// Stop recording
			win.sttRecording = false
			win.sttTranscribing = true
			win.w.Invalidate()

			tempDir := os.TempDir()
			wavPath := filepath.Join(tempDir, "spark_stt.wav")

			err := win.sttRecorder.Stop(wavPath)
			if err != nil {
				win.sttTranscribing = false
				win.ShowToast(fmt.Sprintf("Failed to stop recording: %v", err))
				win.w.Invalidate()
				break
			}

			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				win.sttCancel = cancel
				defer cancel()

				text, err := stt.Transcribe(ctx, win.config.STT, wavPath)
				_ = os.Remove(wavPath) // Clean up temp file

				// Send result to main thread via channel (non-blocking).
				select {
				case win.sttResultCh <- sttResult{text: text, err: err}:
				default:
				}

				// Trigger a frame so the main thread processes the result.
				win.w.Invalidate()
			}()
		} else if win.sttTranscribing {
			// Cancel ongoing transcription
			if win.sttCancel != nil {
				win.sttCancel()
			}
			win.sttTranscribing = false
			win.ShowToast("Transcription cancelled")
			win.w.Invalidate()
		} else {
			// Start recording
			if win.config.STT.APIKey == "" {
				win.ShowToast("STT API key is empty! Set stt.api_key in ~/.spark/config.json")
				break
			}

			win.sttRecorder = stt.NewRecorder()
			err := win.sttRecorder.Start()
			if err != nil {
				win.ShowToast(fmt.Sprintf("Failed to record: %v", err))
				break
			}

			win.sttRecording = true
			win.w.Invalidate()
		}
	}
}

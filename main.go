package main

import (
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/op"
	"gioui.org/unit"
	"yutug.lol/spark/internal/ui"
)

func main() {
	go func() {
		w := new(app.Window)
		w.Option(
			app.Title("spark"),
			app.Size(unit.Dp(1000), unit.Dp(650)),
			app.MinSize(unit.Dp(400), unit.Dp(300)),
			app.Decorated(false),
		)

		win, err := ui.New(w)
		if err != nil {
			log.Fatalf("spark: failed to start terminal: %v", err)
		}
		defer win.ReadyForClose()

		var ops op.Ops
		for {
			ev := w.Event()
			switch e := ev.(type) {
			case app.DestroyEvent:
				if e.Err != nil {
					log.Println("spark: window error:", e.Err)
				}
				os.Exit(0)
			case app.FrameEvent:
				gtx := app.NewContext(&ops, e)
				win.Layout(gtx, w)
				e.Frame(gtx.Ops)
			}
		}
	}()
	app.Main()
}

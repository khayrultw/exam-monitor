package main

import (
	"log"
	"os"
	"sync"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

type AppState struct {
	currentScreen string
	mu            sync.Mutex
}

func NewAppState() AppState {
	return AppState{
		currentScreen: "join",
	}
}

func (state *AppState) swtichScreen(screen string) {
	state.mu.Lock()
	state.currentScreen = screen
	state.mu.Unlock()
}

func main() {
	go func() {
		w := new(app.Window)
		w.Option(app.Title("Hello gio"))
		w.Option(app.Size(unit.Dp(400), unit.Dp(600)))
		if err := run(w); err != nil {
			log.Fatal(err)
			os.Exit(0)
		}
	}()

	app.Main()
}

func run(w *app.Window) error {
	var ops op.Ops
	th := material.NewTheme()
	state := NewAppState()
	dashboard := NewDashboardState(func() {
		state.swtichScreen("join")
	})

	joinView := NewJoinView(func(name string, room int) {
		state.swtichScreen("dashboard")
		dashboard.client.Start(name, room, func() {
			w.Invalidate()
		})
	})

	for {
		event := w.Event()
		switch typ := event.(type) {
		case app.FrameEvent:
			gtx := app.NewContext(&ops, typ)
			switch state.currentScreen {
			case "join":
				layout.Flex{
					Axis: layout.Vertical,
				}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return joinView.Layout(gtx, th)
					}),
				)
			default:
				layout.Flex{
					Axis: layout.Vertical,
				}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return dashboard.Layout(gtx, th)
					}),
				)
			}

			typ.Frame(gtx.Ops)
		case app.DestroyEvent:
			os.Exit(0)
		}
	}

}

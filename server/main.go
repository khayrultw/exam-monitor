package main

import (
	"log"
	"os"
	"sync"
	"time"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type AppState struct {
	currentScreen string
	mu            sync.Mutex
}

func NewAppState() AppState {
	return AppState{
		currentScreen: "home",
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

		w.Option(app.Title("Exam Monitor"))
		w.Option(app.Size(unit.Dp(1000), unit.Dp(700)))

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
	server := NewServer()

	home := NewHomeState(func(room int) {
		state.swtichScreen("dashboard")
		server.Start(room)
	})

	dashboard := NewDashboardState(func() {
		state.swtichScreen("home")
		server.Stop()
	})

	server.studentUtil = dashboard

	var list widget.List
	list.Axis = layout.Vertical

	invalidateTicker := time.NewTicker(time.Second / 4)
	go func() {
		for range invalidateTicker.C {
			w.Invalidate()
		}
	}()

	for {
		event := w.Event()
		switch typ := event.(type) {
		case app.FrameEvent:
			gtx := app.NewContext(&ops, typ)
			switch state.currentScreen {
			case "home":
				layout.Flex{
					Axis: layout.Vertical,
				}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return home.Layout(gtx, th)
					}),
				)
			default:
				layout.Flex{
					Axis: layout.Vertical,
				}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return dashboard.Layout(gtx, th, &list)
					}),
				)
			}

			typ.Frame(gtx.Ops)
		case app.DestroyEvent:
			os.Exit(0)
		}
	}
}

package main

import (
	"image/color"
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

// Custom palette for theming
var AppPalette = material.Palette{
	Bg:         color.NRGBA{R: 255, G: 255, B: 255, A: 255}, // White
	Fg:         color.NRGBA{R: 17, G: 24, B: 39, A: 255},    // gray-900
	ContrastBg: color.NRGBA{R: 37, G: 99, B: 235, A: 255},   // blue-600 (Primary)
	ContrastFg: color.NRGBA{R: 255, G: 255, B: 255, A: 255}, // White
}

// Error color for validation
var ErrorColor = color.NRGBA{R: 220, G: 38, B: 38, A: 255} // red-600

// Disabled colors
var DisabledBg = color.NRGBA{R: 229, G: 231, B: 235, A: 255} // gray-200
var DisabledFg = color.NRGBA{R: 156, G: 163, B: 175, A: 255} // gray-400

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
		w.Option(app.Title("Exam Guard Client"))
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
	th.Palette = AppPalette
	state := NewAppState()

	dashboard := NewDashboardState(func() {
		state.swtichScreen("join")
	}, func() {
		w.Invalidate()
	})

	joinView := NewJoinView(func(sid string, name string, room int) {
		state.swtichScreen("dashboard")
		dashboard.client.Start(sid, name, room, func() {
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

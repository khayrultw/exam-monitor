package main

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type DashboardState struct {
	client  *Client
	BtnStop *widget.Clickable
	Stop    func()
}

func NewDashboardState(stop func()) *DashboardState {
	return &DashboardState{
		client:  NewClient(),
		BtnStop: new(widget.Clickable),
		Stop:    stop,
	}
}

func (d *DashboardState) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if d.BtnStop.Clicked(gtx) {
		d.Stop()
		d.client.Stop()
	}

	if d.client.isConnected.Load() {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(
				gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.H5(th, "Connected").Layout(gtx)
				}),
			)
		})
	} else {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(
				gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.H5(th, "Looking for the server").Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Spacer{Height: unit.Dp(16)}.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					btn := material.Button(th, d.BtnStop, "Stop")
					btn.Background = color.NRGBA{R: 200, G: 50, B: 50, A: 255} // Set button background to a mixed red color
					return btn.Layout(gtx)
				}),
			)
		})
	}
}

package main

import (
	"gioui.org/layout"
	"gioui.org/widget/material"
)

type DashboardState struct {
	client *Client
}

func NewDashboardState() *DashboardState {
	return &DashboardState{
		client: NewClient(),
	}
}

func (d *DashboardState) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
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
			)
		})
	}
}

package main

import (
	"fmt"
	"time"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type DashboardState struct {
	client    *Client
	BtnStop   *widget.Clickable
	BtnRetry  *widget.Clickable
	BtnCancel *widget.Clickable
	Stop      func()
	UpdateUI  func()
	errorMsg  string
}

func NewDashboardState(stop func(), updateUI func()) *DashboardState {
	client := NewClient()
	ds := &DashboardState{
		client:    client,
		BtnStop:   new(widget.Clickable),
		BtnRetry:  new(widget.Clickable),
		BtnCancel: new(widget.Clickable),
		Stop:      stop,
		UpdateUI:  updateUI,
	}

	client.SetCallbacks(
		func() {
			ds.errorMsg = ""
			ds.UpdateUI()
		},
		func(err error) {
			ds.errorMsg = err.Error()
			ds.UpdateUI()
		},
	)

	return ds
}

func (d *DashboardState) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if d.BtnStop.Clicked(gtx) {
		d.Stop()
		d.client.Stop()
	}

	if d.BtnRetry.Clicked(gtx) {
		d.errorMsg = ""
	}

	if d.BtnCancel.Clicked(gtx) {
		d.Stop()
		d.client.Stop()
	}

	if d.client.isConnected.Load() {
		return d.layoutConnected(gtx, th)
	} else {
		return d.layoutSearching(gtx, th)
	}
}

func (d *DashboardState) layoutSearching(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return MaxWidthContainer(gtx, 480, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(
				gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						loader := material.Loader(th)
						return loader.Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Spacer{Height: unit.Dp(16)}.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						msg := material.Body1(th, "Looking for the server…")
						return msg.Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Spacer{Height: unit.Dp(8)}.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						hint := material.Body2(th, "This can take up to 10s.")
						hint.Color = DisabledFg
						hint.TextSize = unit.Sp(12)
						return hint.Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Spacer{Height: unit.Dp(24)}.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if d.errorMsg != "" {
						return layout.Inset{Bottom: 16}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return NewBorderWithColor(ErrorColor).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										errLabel := material.Body2(th, "Could not connect. Please try again.")
										errLabel.Color = ErrorColor
										return errLabel.Layout(gtx)
									})
								})
							})
						})
					}
					return layout.Dimensions{}
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(th, d.BtnCancel, "Cancel")
						btn.Background = DisabledBg
						btn.Color = DisabledFg
						gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
						return btn.Layout(gtx)
					})
				}),
			)
		})
	})
}

func (d *DashboardState) layoutConnected(gtx layout.Context, th *material.Theme) layout.Dimensions {
	lastSent := d.client.GetLastSentTime()
	var timeSinceStr string
	if !lastSent.IsZero() {
		since := time.Since(lastSent)
		if since < time.Second {
			timeSinceStr = "just now"
		} else if since < time.Minute {
			timeSinceStr = fmt.Sprintf("%ds ago", int(since.Seconds()))
		} else {
			timeSinceStr = fmt.Sprintf("%dm ago", int(since.Minutes()))
		}
	} else {
		timeSinceStr = "pending"
	}

	return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return MaxWidthContainer(gtx, 480, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(
				gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						status := material.H6(th, "✓  Screen Sharing Started")
						status.Color = th.Palette.ContrastBg
						return status.Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Spacer{Height: unit.Dp(24)}.Layout(gtx)
				}),
			
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						lastSentLabel := material.Body2(th, fmt.Sprintf("Last sent: %s", timeSinceStr))
						lastSentLabel.Color = DisabledFg
						lastSentLabel.TextSize = unit.Sp(12)
						return lastSentLabel.Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Spacer{Height: unit.Dp(32)}.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(th, d.BtnStop, "Stop")
						btn.Background = ErrorColor
						gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
						return btn.Layout(gtx)
					})
				}),
			)
		})
	})
}

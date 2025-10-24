package main

import (
	"image/color"
	"strconv"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type HomeState struct {
	RoomEditor *widget.Editor
	BtnConnect *widget.Clickable
	OnClick    func(int)
	ErrorText  string
}

func NewHomeState(start func(int)) *HomeState {
	home := HomeState{
		RoomEditor: new(widget.Editor),
		BtnConnect: new(widget.Clickable),
		OnClick:    start,
		ErrorText:  "",
	}

	return &home
}

func (h *HomeState) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if h.BtnConnect.Clicked(gtx) {
		roomText := h.RoomEditor.Text()
		if roomText == "" {
			h.ErrorText = "Room number is required"
		} else {
			room, err := strconv.Atoi(roomText)
			if err != nil {
				h.ErrorText = "Room number must be numeric"
			} else if room <= 0 {
				h.ErrorText = "Room number must be positive"
			} else {
				h.ErrorText = ""
				h.OnClick(room)
			}
		}
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(32), Bottom: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					title := material.H4(th, "Exam Monitor")
					return title.Layout(gtx)
				})
			})
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(
					gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.Body1(th, "Room").Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Spacer{Height: unit.Dp(8)}.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(300)
						gtx.Constraints.Max.X = gtx.Dp(300)
						return TextEditor(th, h.RoomEditor, "Enter room number")(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Spacer{Height: unit.Dp(16)}.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(200)
						btn := material.Button(th, h.BtnConnect, "Connect")
						// Disable button if room is empty
						if h.RoomEditor.Text() == "" {
							btn.Background = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
						}
						return btn.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if h.ErrorText == "" {
							return layout.Dimensions{}
						}
						return layout.Inset{Top: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							errorLabel := material.Body2(th, h.ErrorText)
							errorLabel.Color = color.NRGBA{R: 200, G: 50, B: 50, A: 255}
							return errorLabel.Layout(gtx)
						})
					}),
				)
			})
		}),
	)
}

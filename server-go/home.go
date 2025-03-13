package main

import (
	"strconv"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type HomeState struct {
	RoomEditor *widget.Editor
	BtnStart   *widget.Clickable
	OnClick    func(int)
}

func NewHomeState(start func(int)) *HomeState {
	home := HomeState{
		RoomEditor: new(widget.Editor),
		BtnStart:   new(widget.Clickable),
		OnClick:    start,
	}

	home.RoomEditor.SetText("1234")
	return &home
}

func (h *HomeState) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if h.BtnStart.Clicked(gtx) {
		room, err := strconv.Atoi(h.RoomEditor.Text())
		if err == nil {
			h.OnClick(room)
		}
	}

	return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(
				gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = 600
					return TextEditor(th, h.RoomEditor, "Enter room number")(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Spacer{Height: unit.Dp(16)}.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = 200
					return material.Button(th, h.BtnStart, "Start").Layout(gtx)
				}),
			)
		})
	})
}

package main

import (
	"image/color"
	"strconv"
	"strings"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type JoinView struct {
	IdEditor   *widget.Editor
	NameEditor *widget.Editor
	RoomEditor *widget.Editor
	BtnStart   *widget.Clickable
	OnClick    func(string, string, int)
}

func NewJoinView(start func(string, string, int)) *JoinView {
	joinView := JoinView{
		IdEditor:   new(widget.Editor),
		NameEditor: new(widget.Editor),
		RoomEditor: new(widget.Editor),
		BtnStart:   new(widget.Clickable),
		OnClick:    start,
	}
	return &joinView
}

func (h *JoinView) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	// Check if all fields are non-empty
	idText := strings.TrimSpace(h.IdEditor.Text())
	nameText := strings.TrimSpace(h.NameEditor.Text())
	roomText := strings.TrimSpace(h.RoomEditor.Text())
	allFieldsFilled := idText != "" && nameText != "" && roomText != ""

	if h.BtnStart.Clicked(gtx) && allFieldsFilled {
		room, err := strconv.Atoi(roomText)
		if err == nil {
			h.OnClick(idText, nameText, room)
		}
	}

	return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(
				gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = 600
					return TextEditor(th, h.IdEditor, "Enter student id")(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Spacer{Height: unit.Dp(16)}.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = 600
					return TextEditor(th, h.NameEditor, "Enter your name")(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Spacer{Height: unit.Dp(16)}.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = 600
					return TextEditor(th, h.RoomEditor, "Enter room number")(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Spacer{Height: unit.Dp(16)}.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = 200
					btn := material.Button(th, h.BtnStart, "Start")
					if !allFieldsFilled {
						btn.Background = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
						btn.Color = color.NRGBA{R: 150, G: 150, B: 150, A: 255}
					}
					return btn.Layout(gtx)
				}),
			)
		})
	})
}

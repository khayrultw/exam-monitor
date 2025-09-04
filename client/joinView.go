package main

import (
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
	joinView.IdEditor.SingleLine = true
	joinView.NameEditor.SingleLine = true
	joinView.RoomEditor.SingleLine = true
	return &joinView
}

func (h *JoinView) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if h.BtnStart.Clicked(gtx) {
		room, err := strconv.Atoi(strings.TrimSpace(h.RoomEditor.Text()))
		if err == nil {
			h.OnClick(
				strings.TrimSpace(h.IdEditor.Text()),
				strings.TrimSpace(h.NameEditor.Text()),
				room)
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
					return material.Button(th, h.BtnStart, "Start").Layout(gtx)
				}),
			)
		})
	})
}

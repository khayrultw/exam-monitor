package main

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func TextEditor(th *material.Theme, editor *widget.Editor, hint string) layout.Widget {
	editor.SingleLine = true
	return func(gtx layout.Context) layout.Dimensions {
		return NewBorder().Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Top: 12, Bottom: 12, Left: 8, Right: 8,
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.Editor(th, editor, hint).Layout(gtx)
			})
		})
	}
}

func NewBorder() widget.Border {
	return widget.Border{
		Color:        color.NRGBA{R: 204, G: 204, B: 204, A: 255},
		CornerRadius: unit.Dp(3),
		Width:        unit.Dp(2),
	}
}

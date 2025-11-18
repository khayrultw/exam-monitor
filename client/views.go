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

func TextEditorWithError(th *material.Theme, editor *widget.Editor, hint string, errorMsg string) layout.Widget {
	editor.SingleLine = true
	return func(gtx layout.Context) layout.Dimensions {
		borderColor := color.NRGBA{R: 204, G: 204, B: 204, A: 255}
		if errorMsg != "" {
			borderColor = ErrorColor
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return NewBorderWithColor(borderColor).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{
						Top: 12, Bottom: 12, Left: 8, Right: 8,
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return material.Editor(th, editor, hint).Layout(gtx)
					})
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if errorMsg != "" {
					return layout.Inset{Top: 4, Left: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return ErrorText(th, errorMsg).Layout(gtx)
					})
				}
				return layout.Dimensions{}
			}),
		)
	}
}

func NewBorder() widget.Border {
	return widget.Border{
		Color:        color.NRGBA{R: 204, G: 204, B: 204, A: 255},
		CornerRadius: unit.Dp(3),
		Width:        unit.Dp(2),
	}
}

func NewBorderWithColor(c color.NRGBA) widget.Border {
	return widget.Border{
		Color:        c,
		CornerRadius: unit.Dp(3),
		Width:        unit.Dp(2),
	}
}

func MaxWidthContainer(gtx layout.Context, widthDp float32, child layout.Widget) layout.Dimensions {
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		maxWidth := gtx.Dp(unit.Dp(widthDp))
		if gtx.Constraints.Max.X > maxWidth {
			gtx.Constraints.Max.X = maxWidth
		}
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return child(gtx)
	})
}

func FormRow(gtx layout.Context, label string, th *material.Theme, widget layout.Widget) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body1(th, label)
				lbl.TextSize = unit.Sp(14)
				return lbl.Layout(gtx)
			})
		}),
		layout.Rigid(widget),
	)
}

func ErrorText(th *material.Theme, text string) material.LabelStyle {
	lbl := material.Body2(th, text)
	lbl.Color = ErrorColor
	lbl.TextSize = unit.Sp(12)
	return lbl
}

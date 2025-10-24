package main

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func LayoutViewer(
	gtx layout.Context,
	th *material.Theme,
	student *Student,
	imgCache *ImageCacheManager,
	btnClose *widget.Clickable,
) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return widget.Border{
				Color: color.NRGBA{R: 0, G: 0, B: 0, A: 200},
				Width: unit.Dp(0),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min = gtx.Constraints.Max
				return layout.Dimensions{Size: gtx.Constraints.Max}
			})
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{
						Top: unit.Dp(16), Left: unit.Dp(16), Right: unit.Dp(16), Bottom: unit.Dp(8),
					}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(th, btnClose, "Close")
						btn.Background = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
						return btn.Layout(gtx)
					})
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						if student.Image == nil {
							return material.H6(th, "No image available").Layout(gtx)
						}

						imgOp := imgCache.GetImageOp(student)
						imgSize := student.Image.Bounds().Size()

						maxWidth := gtx.Constraints.Max.X - gtx.Dp(32)
						maxHeight := gtx.Constraints.Max.Y - gtx.Dp(100)

						scaleX := float32(maxWidth) / float32(imgSize.X)
						scaleY := float32(maxHeight) / float32(imgSize.Y)
						scale := scaleX
						if scaleY < scaleX {
							scale = scaleY
						}

						return widget.Image{
							Src:   imgOp,
							Scale: scale,
						}.Layout(gtx)
					})
				}),
			)
		}),
	)
}

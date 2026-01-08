package main

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// Colors for viewer
var (
	overlayColor     = color.NRGBA{R: 245, G: 247, B: 250, A: 255} // Light gray background
	closeButtonBg    = color.NRGBA{R: 239, G: 68, B: 68, A: 255}   // Red
	textDark         = color.NRGBA{R: 30, G: 41, B: 59, A: 255}    // Slate-800
	textMuted        = color.NRGBA{R: 100, G: 116, B: 139, A: 255} // Slate-500
)

func LayoutViewer(
	gtx layout.Context,
	th *material.Theme,
	student *Student,
	imgCache *ImageCacheManager,
	btnClose *widget.Clickable,
) layout.Dimensions {
	paint.FillShape(gtx.Ops, overlayColor, clip.Rect{Max: gtx.Constraints.Max}.Op())

	return layout.UniformInset(unit.Dp(24)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{
						Axis:      layout.Horizontal,
						Alignment: layout.Middle,
						Spacing:   layout.SpaceBetween,
					}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Baseline}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									label := material.H6(th, student.Name)
									label.Color = textDark
									return label.Layout(gtx)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Left: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										label := material.Body1(th, "#"+student.Id)
										label.Color = textMuted
										return label.Layout(gtx)
									})
								}),
							)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							btn := material.Button(th, btnClose, "✕  Close")
							btn.Background = closeButtonBg
							btn.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
							btn.TextSize = unit.Sp(14)
							return btn.Layout(gtx)
						}),
					)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if student.Image == nil {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						label := material.H6(th, "No image available")
						label.Color = textMuted
						return label.Layout(gtx)
					})
				}

				imgOp := imgCache.GetImageOp(student)
				imgSize := student.Image.Bounds().Size()

				availableWidth := gtx.Constraints.Max.X
				availableHeight := gtx.Constraints.Max.Y

				scaleX := float32(availableWidth) / float32(imgSize.X)
				scaleY := float32(availableHeight) / float32(imgSize.Y)
				scale := scaleX
				if scaleY < scaleX {
					scale = scaleY
				}

				scaledWidth := int(float32(imgSize.X) * scale)
				scaledHeight := int(float32(imgSize.Y) * scale)

				offsetX := (availableWidth - scaledWidth) / 2
				offsetY := (availableHeight - scaledHeight) / 2

				return layout.Inset{
					Left: unit.Dp(float32(offsetX) / gtx.Metric.PxPerDp),
					Top:  unit.Dp(float32(offsetY) / gtx.Metric.PxPerDp),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return widget.Image{
						Src:   imgOp,
						Scale: scale,
					}.Layout(gtx)
				})
			}),
		)
	})
}

package main

import (
	"fmt"
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

var (
	topbarBg     = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	topbarBorder = color.NRGBA{R: 226, G: 232, B: 240, A: 255}
	primaryColor = color.NRGBA{R: 59, G: 130, B: 246, A: 255}  // Blue-500
	dangerColor  = color.NRGBA{R: 239, G: 68, B: 68, A: 255}   // Red-500
	neutralColor = color.NRGBA{R: 71, G: 85, B: 105, A: 255}   // Slate-600 (darker)
	controlColor = color.NRGBA{R: 51, G: 65, B: 85, A: 255}    // Slate-700 for +/- buttons
	labelColor   = color.NRGBA{R: 71, G: 85, B: 105, A: 255}   // Slate-600
	countColor   = color.NRGBA{R: 30, G: 41, B: 59, A: 255}    // Slate-800
	badgeBg      = color.NRGBA{R: 219, G: 234, B: 254, A: 255} // Blue-100
	badgeText    = color.NRGBA{R: 30, G: 64, B: 175, A: 255}   // Blue-800
)

func LayoutTopBar(
	gtx layout.Context,
	th *material.Theme,
	studentCount int,
	sortField string,
	sortAsc bool,
	columnsCount int,
	btnSortField *widget.Clickable,
	btnSortToggle *widget.Clickable,
	btnColMinus *widget.Clickable,
	btnColPlus *widget.Clickable,
	btnStop *widget.Clickable,
) layout.Dimensions {
	return layout.Inset{
		Top: unit.Dp(0), Bottom: unit.Dp(8), Left: unit.Dp(8), Right: unit.Dp(8),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		rect := clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)}
		paint.FillShape(gtx.Ops, topbarBg, rect.Op())

		return layout.Inset{
			Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(16), Right: unit.Dp(16),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{
				Axis:      layout.Horizontal,
				Spacing:   layout.SpaceBetween,
				Alignment: layout.Middle,
			}.Layout(
				gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layoutStudentCount(gtx, th, studentCount)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layoutControls(gtx, th, sortField, sortAsc, columnsCount,
						btnSortField, btnSortToggle, btnColMinus, btnColPlus)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					btn := material.Button(th, btnStop, "Stop Session")
					btn.Background = dangerColor
					btn.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
					btn.TextSize = unit.Sp(14)
					return btn.Layout(gtx)
				}),
			)
		})
	})
}

func layoutStudentCount(gtx layout.Context, th *material.Theme, count int) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			label := material.Body1(th, "Students")
			label.Color = labelColor
			label.TextSize = unit.Sp(14)
			return label.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Stack{Alignment: layout.Center}.Layout(gtx,
					layout.Stacked(func(gtx layout.Context) layout.Dimensions {
						macro := func() layout.Dimensions {
							label := material.Body1(th, fmt.Sprintf("%d", count))
							label.TextSize = unit.Sp(14)
							gtx.Constraints.Min = image.Point{}
							return label.Layout(gtx)
						}
						dims := macro()

						padding := gtx.Dp(unit.Dp(8))
						height := gtx.Dp(unit.Dp(24))
						width := dims.Size.X + padding*2
						if width < height {
							width = height
						}

						rect := clip.RRect{
							Rect: image.Rect(0, 0, width, height),
							NE:   12, NW: 12, SE: 12, SW: 12,
						}
						paint.FillShape(gtx.Ops, badgeBg, rect.Op(gtx.Ops))

						return layout.Dimensions{Size: image.Pt(width, height)}
					}),
					layout.Stacked(func(gtx layout.Context) layout.Dimensions {
						label := material.Body1(th, fmt.Sprintf("%d", count))
						label.Color = badgeText
						label.TextSize = unit.Sp(14)
						return label.Layout(gtx)
					}),
				)
			})
		}),
	)
}

func layoutControls(
	gtx layout.Context,
	th *material.Theme,
	sortField string,
	sortAsc bool,
	columnsCount int,
	btnSortField *widget.Clickable,
	btnSortToggle *widget.Clickable,
	btnColMinus *widget.Clickable,
	btnColPlus *widget.Clickable,
) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(
		gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			label := material.Body2(th, "Sort by")
			label.Color = labelColor
			label.TextSize = unit.Sp(13)
			return label.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				btnText := "Name"
				if sortField == "id" {
					btnText = "ID"
				}
				btn := material.Button(th, btnSortField, btnText)
				btn.Background = primaryColor
				btn.TextSize = unit.Sp(13)
				return btn.Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				btnText := "↑"
				if !sortAsc {
					btnText = "↓"
				}
				btn := material.Button(th, btnSortToggle, btnText)
				btn.Background = neutralColor
				btn.TextSize = unit.Sp(14)
				return btn.Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				height := gtx.Dp(unit.Dp(24))
				rect := clip.Rect{Max: image.Pt(1, height)}
				paint.FillShape(gtx.Ops, topbarBorder, rect.Op())
				return layout.Dimensions{Size: image.Pt(1, height)}
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			label := material.Body2(th, "Columns")
			label.Color = labelColor
			label.TextSize = unit.Sp(13)
			return label.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				btn := material.Button(th, btnColMinus, "−")
				btn.Background = controlColor
				btn.TextSize = unit.Sp(16)
				return btn.Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				label := material.Body1(th, fmt.Sprintf("%d", columnsCount))
				label.Color = countColor
				label.TextSize = unit.Sp(14)
				return label.Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			btn := material.Button(th, btnColPlus, "+")
			btn.Background = controlColor
			btn.TextSize = unit.Sp(16)
			return btn.Layout(gtx)
		}),
	)
}

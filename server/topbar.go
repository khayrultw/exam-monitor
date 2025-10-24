package main

import (
	"fmt"
	"image/color"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
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
		Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(8), Right: unit.Dp(8),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return widget.Border{
			Color: color.NRGBA{R: 220, G: 220, B: 220, A: 255},
			Width: unit.Dp(2),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(8), Right: unit.Dp(8),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween, Alignment: layout.Middle}.Layout(
					gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.Body1(th, fmt.Sprintf("Total: %d", studentCount)).Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layoutSortAndColumnControls(gtx, th, sortField, sortAsc, columnsCount,
							btnSortField, btnSortToggle, btnColMinus, btnColPlus)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(th, btnStop, "Stop")
						btn.Background = color.NRGBA{R: 200, G: 50, B: 50, A: 255}
						return btn.Layout(gtx)
					}),
				)
			})
		})
	})
}

func layoutSortAndColumnControls(
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
			return material.Body1(th, "Sort:").Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				btnText := "Name"
				if sortField == "id" {
					btnText = "ID"
				}
				return material.Button(th, btnSortField, btnText).Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				btnText := "Asc ↑"
				if !sortAsc {
					btnText = "Desc ↓"
				}
				return material.Button(th, btnSortToggle, btnText).Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.Body1(th, "Columns:").Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.Button(th, btnColMinus, "-").Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.Body1(th, fmt.Sprintf("%d", columnsCount)).Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Button(th, btnColPlus, "+").Layout(gtx)
		}),
	)
}

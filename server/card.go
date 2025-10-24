package main

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func StudentCard(gtx layout.Context, th *material.Theme, student *Student, width int, imgCache *ImageCacheManager) layout.Dimensions {
	return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return material.Clickable(gtx, student.Clickable, func(gtx layout.Context) layout.Dimensions {
			return widget.Border{
				Color: color.NRGBA{A: 64, R: 180, G: 180, B: 180},
				Width: unit.Dp(2),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{
					Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(8), Right: unit.Dp(8),
				}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(
						gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layoutStudentImage(gtx, student, width, imgCache)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layoutStudentInfo(gtx, th, student)
						}),
					)
				})
			})
		})
	})
}

func layoutStudentImage(gtx layout.Context, student *Student, width int, imgCache *ImageCacheManager) layout.Dimensions {
	if student.Image == nil {
		return layout.Dimensions{Size: image.Pt(width, width*3/4)}
	}
	imgOp := imgCache.GetImageOp(student)
	imgSize := student.Image.Bounds().Size()
	scale := float32(width) / float32(imgSize.X)
	return widget.Image{
		Src:   imgOp,
		Scale: scale,
	}.Layout(gtx)
}

func layoutStudentInfo(gtx layout.Context, th *material.Theme, student *Student) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal}.Layout(
			gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				label := material.Body1(th, student.Name)
				label.MaxLines = 1
				return label.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Spacer{Width: unit.Dp(8)}.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				label := material.Body1(th, "#"+student.Id)
				label.Color = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
				label.MaxLines = 1
				return label.Layout(gtx)
			}),
		)
	})
}

func CreateStudentGrid(gtx layout.Context, th *material.Theme, students []*Student, rowIndex int, col int, imgCache *ImageCacheManager) []layout.FlexChild {
	var row []layout.FlexChild
	start := rowIndex * col
	end := start + col
	if end > len(students) {
		end = len(students)
	}
	width := (gtx.Constraints.Max.X - 2*col*8) / col

	for i := start; i < end; i++ {
		item := students[i]
		row = append(row, layout.Flexed(1.0/float32(col), func(gtx layout.Context) layout.Dimensions {
			return StudentCard(gtx, th, item, width, imgCache)
		}))
	}

	// Fill empty slots
	for len(row) < col {
		row = append(row, layout.Flexed(1.0/float32(col), func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{}
		}))
	}

	return row
}

package main

import (
	"fmt"
	"image"
	"sync"

	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type Student struct {
	Id    int
	Name  string
	Image image.Image
}

type DashboardState struct {
	Students map[int]*Student
	NextId   int
	BtnStop  *widget.Clickable
	Stop     func()
	mu       sync.Mutex
}

const col = 2

func NewDashboardState(stop func()) *DashboardState {
	return &DashboardState{
		Students: make(map[int]*Student),
		NextId:   0,
		BtnStop:  new(widget.Clickable),
		Stop:     stop,
	}
}

func (ds *DashboardState) AddStudent(name string) *Student {

	student := &Student{
		Id:   ds.NextId,
		Name: name,
	}

	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.NextId += 1
	ds.Students[student.Id] = student

	return student
}

func (ds *DashboardState) removeStudent(id int) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	delete(ds.Students, id)
}

func (ds *DashboardState) getStudentsAsSlice() []*Student {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	students := make([]*Student, 0, len(ds.Students))
	for _, student := range ds.Students {
		students = append(students, student)
	}
	return students
}

func (ds *DashboardState) Layout(gtx layout.Context, th *material.Theme, list *widget.List) layout.Dimensions {
	if ds.BtnStop.Clicked(gtx) {
		ds.Stop()
		ds.mu.Lock()
		defer ds.mu.Unlock()
		ds.Students = make(map[int]*Student, 0)
	}
	itemCount := len(ds.Students)/col + int(len(ds.Students)%col)
	return layout.Flex{Axis: layout.Vertical}.Layout(
		gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Top: 8, Bottom: 8, Left: 16, Right: 16,
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(
					gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.H5(th, fmt.Sprintf("Connected students %d", len(ds.Students))).Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.Button(th, ds.BtnStop, "Stop").Layout(gtx)
					}),
				)
			})
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return list.Layout(gtx, itemCount, func(gtx layout.Context, index int) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, ds.CreateRow(gtx, th, index)...)
			})
		}),
	)
}

func (ds *DashboardState) CreateRow(gtx layout.Context, th *material.Theme, rowIndex int) []layout.FlexChild {
	var row []layout.FlexChild
	students := ds.getStudentsAsSlice()
	start := rowIndex * col
	end := min(start+col, len(students))
	width := (gtx.Constraints.Max.X - 32) / col // Get equal width for each item

	for i := start; i < end; i++ {
		item := students[i]
		row = append(row, layout.Flexed(1.0/float32(col), func(gtx layout.Context) layout.Dimensions {
			return StudentCard(th, *item, width)(gtx)
		}))
	}

	// Fill empty slots in case items are not exactly multiple of 3
	for len(row) < col {
		row = append(row, layout.Flexed(1.0/float32(col), func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{} // Empty space
		}))
	}

	return row
}

func StudentCard(th *material.Theme, item Student, width int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(
				gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Body1(th, item.Name).Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if item.Image == nil {
						return layout.Dimensions{}
					}
					imgOp := paint.NewImageOp(item.Image)
					imgSize := item.Image.Bounds().Size()
					scale := float32(width) / float32(imgSize.X)
					return widget.Image{
						Src:   imgOp,
						Scale: scale,
					}.Layout(gtx)
				}),
			)
		})
	}
}

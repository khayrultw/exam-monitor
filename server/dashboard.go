package main

import (
	"fmt"
	"image"
	"image/color"
	"sort"
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

const col = 3

func NewDashboardState(stop func()) *DashboardState {
	return &DashboardState{
		Students: make(map[int]*Student),
		NextId:   0,
		BtnStop:  new(widget.Clickable),
		Stop:     stop,
	}
}

func (ds *DashboardState) removeDuplicatesByName() {
	highestIDByName := make(map[string]int)

	for _, student := range ds.Students {
		if currentMaxID, exists := highestIDByName[student.Name]; !exists || student.Id > currentMaxID {
			highestIDByName[student.Name] = student.Id
		}
	}

	for id, student := range ds.Students {
		if highestIDByName[student.Name] != student.Id {
			delete(ds.Students, id)
		}
	}
}

func (ds *DashboardState) AddStudent(name string) int {

	student := &Student{
		Id:   ds.NextId,
		Name: name,
	}

	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.NextId += 1
	ds.Students[student.Id] = student

	return student.Id
}

func (ds *DashboardState) isExists(id int) bool {
	_, ok := ds.Students[id]
	return ok
}

func (ds *DashboardState) RemoveStudent(id int) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	delete(ds.Students, id)
}

func (ds *DashboardState) UpdateImage(id int, img image.Image) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	student, ok := ds.Students[id]
	if !ok {
		return
	}
	student.Image = img
}

func (ds *DashboardState) UpdateName(id int, name string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	student, ok := ds.Students[id]
	if !ok {
		return
	}
	student.Name = name
}

func (ds *DashboardState) getStudentsAsSlice() []*Student {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ds.removeDuplicatesByName()

	keys := make([]int, 0, len(ds.Students))
	for k := range ds.Students {
		keys = append(keys, k)
	}
	sort.Ints(keys) // Sorting the keys in ascending order

	students := make([]*Student, 0, len(ds.Students))

	for _, k := range keys {
		students = append(students, ds.Students[k])
	}
	return students
}

func (ds *DashboardState) Layout(gtx layout.Context, th *material.Theme, list *widget.List) layout.Dimensions {

	if ds.BtnStop.Clicked(gtx) {
		ds.Stop()
		ds.mu.Lock()
		defer ds.mu.Unlock()
		ds.Students = make(map[int]*Student, 0)
		return layout.Dimensions{}
	}

	students := ds.getStudentsAsSlice()
	itemCount := len(students)/col + int(len(students)%col)
	return layout.Flex{Axis: layout.Vertical}.Layout(
		gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Top: 8, Bottom: 8, Left: 16, Right: 16,
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(
					gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.H5(th, fmt.Sprintf("Connected students %d", len(students))).Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(th, ds.BtnStop, "Stop")
						btn.Background = color.NRGBA{R: 200, G: 50, B: 50, A: 255} // Set button background to a mixed red color
						return btn.Layout(gtx)
					}),
				)
			})
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return list.Layout(gtx, itemCount, func(gtx layout.Context, index int) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, ds.CreateRow(gtx, th, students, index)...)
			})
		}),
	)
}

func (ds *DashboardState) CreateRow(gtx layout.Context, th *material.Theme, students []*Student, rowIndex int) []layout.FlexChild {
	var row []layout.FlexChild
	start := rowIndex * col
	end := min(start+col, len(students))
	width := (gtx.Constraints.Max.X - 2*col*8) / col // Get equal width for each item

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

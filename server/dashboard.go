package main

import (
	"fmt"
	"image"
	"image/color"
	"sort"
	"strconv"
	"strings"
	"sync"

	"gioui.org/layout"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type Student struct {
	Id    string
	Name  string
	Image image.Image
}

type DashboardState struct {
	colEditor *widget.Editor
	Students  map[string]*Student
	BtnStop   *widget.Clickable
	Stop      func()
	mu        sync.Mutex
}

func NewDashboardState(stop func()) *DashboardState {
	return &DashboardState{
		colEditor: new(widget.Editor),
		Students:  make(map[string]*Student),
		BtnStop:   new(widget.Clickable),
		Stop:      stop,
	}
}

func (ds *DashboardState) AddStudent(id, name string) {

	student := &Student{
		Id:   id,
		Name: name,
	}

	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.Students[student.Id] = student
}

func (ds *DashboardState) isExists(id string) bool {
	_, ok := ds.Students[id]
	return ok
}

func (ds *DashboardState) RemoveStudent(id string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	delete(ds.Students, id)
}

func (ds *DashboardState) UpdateImage(id string, img image.Image) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	student, ok := ds.Students[id]
	if !ok {
		return
	}
	student.Image = img
}

func (ds *DashboardState) UpdateName(id string, name string) {
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

	students := make([]*Student, 0, len(ds.Students))
	for _, student := range ds.Students {
		students = append(students, student)
	}

	sort.Slice(students, func(i, j int) bool {
		return students[i].Name < students[j].Name
	})

	return students
}

func (ds *DashboardState) Layout(gtx layout.Context, th *material.Theme, list *widget.List) layout.Dimensions {

	if ds.BtnStop.Clicked(gtx) {
		ds.Stop()
		ds.mu.Lock()
		defer ds.mu.Unlock()
		ds.Students = make(map[string]*Student, 0)
		return layout.Dimensions{}
	}

	col, err := strconv.Atoi(strings.TrimSpace(ds.colEditor.Text()))
	if err != nil || col <= 0 {
		col = 3
	}

	students := ds.getStudentsAsSlice()
	itemCount := len(students)/col + int(len(students)%col)
	return layout.Flex{Axis: layout.Vertical}.Layout(
		gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Top: 8, Bottom: 8, Left: 8, Right: 16,
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(
					gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.H6(th, fmt.Sprintf("Total: %d", len(students))).Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{
							Axis:      layout.Horizontal,
							Alignment: layout.Middle, // Center vertically
						}.Layout(
							gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return material.Body1(th, "Columns: ").Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								gtx.Constraints.Min.X = 50
								return TextEditor(th, ds.colEditor, "3")(gtx)
							}),
						)
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
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, ds.CreateRow(gtx, th, students, index, col)...)
			})
		}),
	)
}

func (ds *DashboardState) CreateRow(gtx layout.Context, th *material.Theme, students []*Student, rowIndex int, col int) []layout.FlexChild {
	var row []layout.FlexChild
	start := rowIndex * col
	end := min(start+col, len(students))
	width := (gtx.Constraints.Max.X - 2*col*8) / col

	for i := start; i < end; i++ {
		item := students[i]
		row = append(row, layout.Flexed(1.0/float32(col), func(gtx layout.Context) layout.Dimensions {
			return StudentCard(th, *item, width)(gtx)
		}))
	}

	// Fill empty slots in case items are not exactly multiple of 3
	for len(row) < col {
		row = append(row, layout.Flexed(1.0/float32(col), func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{}
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

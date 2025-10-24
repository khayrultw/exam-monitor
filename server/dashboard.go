package main

import (
	"image"

	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type DashboardState struct {
	studentManager  *StudentManager
	imgCache        *ImageCacheManager
	BtnStop         *widget.Clickable
	BtnColMinus     *widget.Clickable
	BtnColPlus      *widget.Clickable
	BtnSortToggle   *widget.Clickable
	BtnSortField    *widget.Clickable
	BtnViewerClose  *widget.Clickable
	Stop            func()
	columnsCount    int
	viewerOpen      bool
	viewerStudentID string
}

func NewDashboardState(stop func()) *DashboardState {
	return &DashboardState{
		studentManager:  NewStudentManager(),
		imgCache:        NewImageCacheManager(),
		BtnStop:         new(widget.Clickable),
		BtnColMinus:     new(widget.Clickable),
		BtnColPlus:      new(widget.Clickable),
		BtnSortToggle:   new(widget.Clickable),
		BtnSortField:    new(widget.Clickable),
		BtnViewerClose:  new(widget.Clickable),
		Stop:            stop,
		columnsCount:    3,
		viewerOpen:      false,
		viewerStudentID: "",
	}
}

// Interface methods for server integration
func (ds *DashboardState) AddStudent(id, name string) {
	ds.studentManager.Add(id, name)
}

func (ds *DashboardState) isExists(id string) bool {
	return ds.studentManager.Exists(id)
}

func (ds *DashboardState) RemoveStudent(id string) {
	ds.studentManager.Remove(id)
	ds.imgCache.Remove(id)
}

func (ds *DashboardState) UpdateImage(id string, img image.Image) {
	ds.studentManager.UpdateImage(id, img)
}

func (ds *DashboardState) UpdateName(id string, name string) {
	ds.studentManager.UpdateName(id, name)
}

func (ds *DashboardState) Layout(gtx layout.Context, th *material.Theme, list *widget.List) layout.Dimensions {
	ds.handleButtonClicks(gtx)

	students := ds.studentManager.GetSorted()

	for _, student := range students {
		if student.Clickable.Clicked(gtx) {
			ds.viewerOpen = true
			ds.viewerStudentID = student.Id
		}
	}

	if ds.viewerOpen {
		viewerStudent := ds.studentManager.GetByID(ds.viewerStudentID)
		if viewerStudent == nil {
			ds.viewerOpen = false
		} else {
			return LayoutViewer(gtx, th, viewerStudent, ds.imgCache, ds.BtnViewerClose)
		}
	}

	return ds.layoutDashboard(gtx, th, list, students)
}

func (ds *DashboardState) handleButtonClicks(gtx layout.Context) {
	if ds.BtnStop.Clicked(gtx) {
		ds.Stop()
		ds.studentManager.Clear()
		ds.imgCache.Clear()
		ds.viewerOpen = false
	}

	if ds.BtnColMinus.Clicked(gtx) && ds.columnsCount > 1 {
		ds.columnsCount--
	}

	if ds.BtnColPlus.Clicked(gtx) && ds.columnsCount < 8 {
		ds.columnsCount++
	}

	if ds.BtnSortToggle.Clicked(gtx) {
		ds.studentManager.ToggleSortDirection()
	}

	if ds.BtnSortField.Clicked(gtx) {
		if ds.studentManager.GetSortField() == "name" {
			ds.studentManager.SetSortField("id")
		} else {
			ds.studentManager.SetSortField("name")
		}
	}

	if ds.BtnViewerClose.Clicked(gtx) {
		ds.viewerOpen = false
	}
}

func (ds *DashboardState) layoutDashboard(gtx layout.Context, th *material.Theme, list *widget.List, students []*Student) layout.Dimensions {
	col := ds.columnsCount
	itemCount := (len(students) + col - 1) / col

	return layout.Flex{Axis: layout.Vertical}.Layout(
		gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return LayoutTopBar(
				gtx, th,
				ds.studentManager.Count(),
				ds.studentManager.GetSortField(),
				ds.studentManager.IsSortAscending(),
				ds.columnsCount,
				ds.BtnSortField,
				ds.BtnSortToggle,
				ds.BtnColMinus,
				ds.BtnColPlus,
				ds.BtnStop,
			)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return list.Layout(gtx, itemCount, func(gtx layout.Context, index int) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(
					gtx,
					CreateStudentGrid(gtx, th, students, index, col, ds.imgCache)...,
				)
			})
		}),
	)
}

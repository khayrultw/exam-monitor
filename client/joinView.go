package main

import (
	"strconv"
	"strings"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type JoinView struct {
	IdEditor   *widget.Editor
	NameEditor *widget.Editor
	RoomEditor *widget.Editor
	BtnStart   *widget.Clickable
	OnClick    func(string, string, int)

	// Validation errors
	idError   string
	nameError string
	roomError string

	// Track submit attempt
	submitAttempted bool
}

func NewJoinView(start func(string, string, int)) *JoinView {
	joinView := JoinView{
		IdEditor:   new(widget.Editor),
		NameEditor: new(widget.Editor),
		RoomEditor: new(widget.Editor),
		BtnStart:   new(widget.Clickable),
		OnClick:    start,
	}

	// Set up numeric filter for Room editor
	joinView.RoomEditor.Filter = "0123456789"
	joinView.RoomEditor.MaxLen = 6

	// Enable submit on all editors
	joinView.IdEditor.Submit = true
	joinView.NameEditor.Submit = true
	joinView.RoomEditor.Submit = true

	// Load saved data if available
	if data, err := LoadFormData(); err == nil && data != nil {
		joinView.IdEditor.SetText(data.StudentID)
		joinView.NameEditor.SetText(data.Name)
		joinView.RoomEditor.SetText(data.Room)
	}

	return &joinView
}

func (h *JoinView) validate() bool {
	h.idError = ""
	h.nameError = ""
	h.roomError = ""

	valid := true

	// Validate Student ID
	if strings.TrimSpace(h.IdEditor.Text()) == "" {
		h.idError = "Student ID is required."
		valid = false
	}

	// Validate Name
	if strings.TrimSpace(h.NameEditor.Text()) == "" {
		h.nameError = "Name is required."
		valid = false
	}

	// Validate Room
	roomText := strings.TrimSpace(h.RoomEditor.Text())
	if roomText == "" {
		h.roomError = "Room is required."
		valid = false
	} else if roomNum, err := strconv.Atoi(roomText); err != nil {
		h.roomError = "Room must be a valid number."
		valid = false
	} else if roomNum <= 0 {
		h.roomError = "Room must be a positive number."
		valid = false
	}

	return valid
}

func (h *JoinView) isValid() bool {
	if strings.TrimSpace(h.IdEditor.Text()) == "" {
		return false
	}
	if strings.TrimSpace(h.NameEditor.Text()) == "" {
		return false
	}
	roomText := strings.TrimSpace(h.RoomEditor.Text())
	if roomText == "" {
		return false
	}
	roomNum, err := strconv.Atoi(roomText)
	if err != nil {
		return false
	}
	if roomNum <= 0 {
		return false
	}
	return true
}

func (h *JoinView) handleSubmit() {
	h.submitAttempted = true
	if h.validate() {
		room, _ := strconv.Atoi(strings.TrimSpace(h.RoomEditor.Text()))
		studentID := strings.TrimSpace(h.IdEditor.Text())
		name := strings.TrimSpace(h.NameEditor.Text())

		// Save form data for next time
		SaveFormData(studentID, name, strings.TrimSpace(h.RoomEditor.Text()))

		h.OnClick(studentID, name, room)
	}
}

func (h *JoinView) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	// Handle button click
	if h.BtnStart.Clicked(gtx) {
		h.handleSubmit()
	}

	// Show errors only after submit attempt
	var idErr, nameErr, roomErr string
	if h.submitAttempted {
		idErr = h.idError
		nameErr = h.nameError
		roomErr = h.roomError
	}

	return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return MaxWidthContainer(gtx, 480, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(
				gtx,
				// Subtitle
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						subtitle := material.Body1(th, "Please enter your details to join:")
						subtitle.TextSize = unit.Sp(14)
						return subtitle.Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Spacer{Height: unit.Dp(24)}.Layout(gtx)
				}),
				// Student ID
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return FormRow(gtx, "Student ID", th, func(gtx layout.Context) layout.Dimensions {
						return TextEditorWithError(th, h.IdEditor, "Enter student id", idErr)(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Spacer{Height: unit.Dp(16)}.Layout(gtx)
				}),
				// Name
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return FormRow(gtx, "Name", th, func(gtx layout.Context) layout.Dimensions {
						return TextEditorWithError(th, h.NameEditor, "Enter your name", nameErr)(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Spacer{Height: unit.Dp(16)}.Layout(gtx)
				}),
				// Room
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return FormRow(gtx, "Room (digits only)", th, func(gtx layout.Context) layout.Dimensions {
						return TextEditorWithError(th, h.RoomEditor, "Enter room number", roomErr)(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Spacer{Height: unit.Dp(24)}.Layout(gtx)
				}),
				// Start button
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(th, h.BtnStart, "Start")

						// Apply disabled style if invalid
						if !h.isValid() {
							btn.Background = DisabledBg
							btn.Color = DisabledFg
						}

						gtx.Constraints.Min.X = gtx.Dp(unit.Dp(200))
						return btn.Layout(gtx)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Spacer{Height: unit.Dp(16)}.Layout(gtx)
				}),
				// Footer
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						footer := material.Body2(th, "v1.0 • Your institution's acceptable use policy applies")
						footer.TextSize = unit.Sp(10)
						footer.Color = DisabledFg
						return footer.Layout(gtx)
					})
				}),
			)
		})
	})
}

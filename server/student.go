package main

import (
	"time"

	"image"

	"gioui.org/widget"
)

type Student struct {
	Id        string
	Name      string
	Image     image.Image
	ImagePtr  uintptr
	Timestamp time.Time
	Clickable *widget.Clickable
}

func NewStudent(id, name string) *Student {
	return &Student{
		Id:        id,
		Name:      name,
		Clickable: new(widget.Clickable),
		Timestamp: time.Now(),
	}
}

func (s *Student) UpdateImage(img image.Image) {
	s.Image = img
	s.ImagePtr = uintptr(0)
	if img != nil {
		s.ImagePtr = uintptr(img.Bounds().Min.X)<<32 | uintptr(img.Bounds().Min.Y)
	}
	s.Timestamp = time.Now()
}

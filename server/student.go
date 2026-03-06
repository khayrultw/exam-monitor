package main

import (
	"image"
	"sync/atomic"

	"gioui.org/widget"
)

var globalImageVersion atomic.Uint64

type Student struct {
	Id           string
	Name         string
	Image        image.Image
	ImageVersion uint64
	Clickable    *widget.Clickable
}

func NewStudent(id, name string) *Student {
	return &Student{
		Id:        id,
		Name:      name,
		Clickable: new(widget.Clickable),
	}
}

func (s *Student) UpdateImage(img image.Image) {
	s.Image = img
	s.ImageVersion = globalImageVersion.Add(1)
}

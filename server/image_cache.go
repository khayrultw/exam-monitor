package main

import (
	"time"

	"gioui.org/op/paint"
)

type ImageCache struct {
	op        paint.ImageOp
	imagePtr  uintptr
	timestamp time.Time
}

type ImageCacheManager struct {
	cache map[string]ImageCache
}

func NewImageCacheManager() *ImageCacheManager {
	return &ImageCacheManager{
		cache: make(map[string]ImageCache),
	}
}

func (icm *ImageCacheManager) GetImageOp(student *Student) paint.ImageOp {
	if student.Image == nil {
		return paint.ImageOp{}
	}

	cached, ok := icm.cache[student.Id]
	if ok && cached.imagePtr == student.ImagePtr && cached.timestamp == student.Timestamp {
		return cached.op
	}

	// Create new image op
	imgOp := paint.NewImageOp(student.Image)
	icm.cache[student.Id] = ImageCache{
		op:        imgOp,
		imagePtr:  student.ImagePtr,
		timestamp: student.Timestamp,
	}
	return imgOp
}

func (icm *ImageCacheManager) Remove(id string) {
	delete(icm.cache, id)
}

func (icm *ImageCacheManager) Clear() {
	icm.cache = make(map[string]ImageCache)
}

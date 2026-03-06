package main

import (
	"gioui.org/op/paint"
)

type ImageCache struct {
	op      paint.ImageOp
	version uint64
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
	if ok && cached.version == student.ImageVersion {
		return cached.op
	}

	// Create new image op
	imgOp := paint.NewImageOp(student.Image)
	icm.cache[student.Id] = ImageCache{
		op:      imgOp,
		version: student.ImageVersion,
	}
	return imgOp
}

func (icm *ImageCacheManager) Remove(id string) {
	delete(icm.cache, id)
}

func (icm *ImageCacheManager) Clear() {
	icm.cache = make(map[string]ImageCache)
}

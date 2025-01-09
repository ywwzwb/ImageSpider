package interfaces

import "ywwzwb/imagespider/models"

const DBServiceID ServiceID = "IDBService"

type IDBService interface {
	InitSource(id string) error
	GetMeta(id, source string) (*models.ImageMeta, bool)
	InsertMeta(meta models.ImageMeta) error
	GetMetaLocalPathNULL(source string, maxSize int) []models.ImageMeta
	UpdateLocalPathForMeta(meta models.ImageMeta) error

	ListNotGroupTags(source string, offset, limit int64) (*models.TagList, error)
	ListDownloadedImageOfTags(source string, tags []string, offset, limit int64) (*models.ImageList, error)
	GetImageMeta(source string, id string) (*models.ImageMeta, error)
}

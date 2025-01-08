package interfaces

import "ywwzwb/imagespider/models"

const IDBServiceID ServiceID = "IDBService"

type IDBService interface {
	InitSource(id string) error
	GetMeta(id, source string) (*models.ImageMeta, bool)
	InsertMeta(meta models.ImageMeta) error
	GetMetaWithoutLocalPath(source string, maxSize int) []models.ImageMeta
	UpdateLocalPathForMeta(meta models.ImageMeta) error

	ListNotGroupTags(source string, offset, limit int64) (*models.TagList, error)
	ListImageOFTags(source string, tags []string, offset, limit int64) (*models.ImageList, error)
}

package interfaces

import (
	"errors"
	"time"

	"ywwzwb/imagespider/models"
)

// DBServiceID 数据库服务 ID
const DBServiceID ServiceID = "DBService"

// ErrNotFound 记录不存在
var ErrNotFound = errors.New("not found")

type IDBService interface {
	InitSource(id string) error
	GetMeta(id, source string) (*models.ImageMeta, bool)
	InsertMeta(meta models.ImageMeta) error
	GetMetaLocalPathNULL(source string, maxSize int) []models.ImageMeta
	UpdateLocalPathForMeta(meta models.ImageMeta) error
	// RecordDownloadFailure 记录一次下载失败；累计失败次数达到 maxAttempts 后标记为永久失败（空路径），
	// 否则保留 local_path = NULL 并在 retryAfter 之后重试。返回累计失败次数和是否永久失败。
	RecordDownloadFailure(source string, id string, postTime time.Time, maxAttempts int, retryAfter time.Duration) (attempts int, permanent bool, err error)

	ListNotGroupTags(source string, offset, limit int64) (*models.TagList, error)
	/**
	* 例举图片
	* @ params: source 源
	* @ params: tags 要显示的图片标签, 空表示所有标签
	* @ params: status 图片破损情况, 空表示显示所有图片
	* @ params: limit 分页参数
	* @ params: offset 分页参数
	 */
	ListDownloadedImage(source string, tags []string, status []models.ImageIntegrityStatus, offset, limit int64) (*models.ImageList, error)
	ListDownloadedImagesWithUnknownStatus(source string, maxSize int) (*models.ImageList, error)
	GetImageMeta(source string, id string) (*models.ImageMeta, error)
	DeleteImageFile(source string, id string) error
	DeleteImageRecord(source string, id string) error
	UpdateImageIntegrityStatus(source string, id string, status models.ImageIntegrityStatus) error
	SetTagCover(source string, tag string, imageID string) error
}

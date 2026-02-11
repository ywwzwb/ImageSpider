package models

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"
)

// ImageIntegrityStatus represents the integrity status of an image
// 0 = unknown, 1 = good, -1 = bad
// Using smallint in database for space efficiency
// Null in database also represents unknown (for backward compatibility)
type ImageIntegrityStatus int16

const (
	ImageIntegrityUnknown ImageIntegrityStatus = 0  // 未知状态（默认值）
	ImageIntegrityGood    ImageIntegrityStatus = 1  // 正常
	ImageIntegrityBad     ImageIntegrityStatus = -1 // 异常
)

type ImageMeta struct {
	ID               string               `json:"id"`
	Tags             []string             `json:"tags"`
	LocalPath        *string              `json:"localPath"`
	ImageURL         string               `json:"imageURL"`
	PostTime         time.Time            `json:"postTime"`
	SourceID         string               `json:"sourceId"`
	IntegrityStatus  ImageIntegrityStatus `json:"integrityStatus"` // 图片完整性状态 (0=未知, 1=好图, -1=坏图)
}

func (i *ImageMeta) Hash() string {
	id := fmt.Sprintf("%s-%s", i.SourceID, i.ID)
	md5 := md5.Sum([]byte(id))
	return hex.EncodeToString(md5[:])
}

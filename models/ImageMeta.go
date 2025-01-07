package models

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"
)

type ImageMeta struct {
	ID        string
	Tags      []string
	LocalPath *string
	ImageURL  string
	PostTime  time.Time
	SourceID  string
}

func (i *ImageMeta) Hash() string {
	id := fmt.Sprintf("%s-%s", i.SourceID, i.ID)
	md5 := md5.Sum([]byte(id))
	return hex.EncodeToString(md5[:])
}

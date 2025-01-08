package models

type ImageList struct {
	ImageList  []ImageMeta `json:"imageList" yaml:"imageList"`
	TotalCount int         `json:"totalCount" yaml:"totalCount"`
}

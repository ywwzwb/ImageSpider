package models

type TagInfo struct {
	Tag   string `json:"tag" yaml:"tag"`
	Count int    `json:"count" yaml:"count"`
}
type TagList struct {
	TagList    []TagInfo `json:"tagList" yaml:"tagList"`
	TotalCount int       `json:"totalCount" yaml:"totalCount"`
}
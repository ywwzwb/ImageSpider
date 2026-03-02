package config

// ThumbnailConfig 缩略图配置
type ThumbnailConfig struct {
	// Width 目标宽度，0 表示自适应
	Width int `json:"width" yaml:"width"`
	// Height 目标高度，0 表示自适应（width 和 height 至少一个非零）
	Height int `json:"height" yaml:"height"`
	// Quality 输出质量 (1-100)，0 表示使用默认值
	Quality int `json:"quality" yaml:"quality"`
	// Suffix 缩略图文件后缀，例如 "@320"
	Suffix string `json:"suffix" yaml:"suffix"`
	// Format 输出格式，空字符串表示与原图相同
	Format string `json:"format" yaml:"format"`
}

type ImageConvertConfig struct {
	// OutputFormat 输出图片格式，例如 "avif", "webp", "jpg"
	OutputFormat string `json:"outputFormat" yaml:"outputFormat"`
	// Quality 图片质量 (1-100)
	Quality int `json:"quality" yaml:"quality"`
	// LosslessModeEnabled 是否启用无损模式
	LosslessModeEnabled bool `json:"losslessModeEnabled" yaml:"losslessModeEnabled"`
	// Thumbnails 缩略图配置列表（可生成多种尺寸）
	Thumbnails []ThumbnailConfig `json:"thumbnails" yaml:"thumbnails"`
}

package interfaces

import "ywwzwb/imagespider/models/config"

// ImageConvertServiceID 图片转换服务 ID
const ImageConvertServiceID ServiceID = "ImageConvert"

// IImageConvertService 图片转换服务接口
type IImageConvertService interface {
	// ConvertImage 转换图片格式
	// input: 输入文件路径
	// output: 输出文件路径（根据后缀确定目标格式）
	// quality: 输出质量 (1-100)，0 表示使用默认值
	ConvertImage(input, output string, quality int) error

	// GenerateThumbnail 生成缩略图
	// input: 输入文件路径
	// output: 输出文件路径
	// width: 目标宽度，0 表示自适应
	// height: 目标高度，0 表示自适应（width 和 height 至少一个非零）
	GenerateThumbnail(input, output string, width, height int, options ThumbnailOptions) error

	// GetDefaultOutputFormat 获取默认输出格式
	GetDefaultOutputFormat() string

	// GetDefaultQuality 获取默认质量
	GetDefaultQuality() int

	// GetThumbnailConfigs 获取缩略图配置列表
	GetThumbnailConfigs() []config.ThumbnailConfig
}

// ThumbnailOptions 缩略图生成选项
type ThumbnailOptions struct {
	// Quality 输出质量 (1-100)，0 表示使用默认值
	Quality int
}

package interfaces

import "ywwzwb/imagespider/models/config"

// ImageDownloaderServiceID 图片下载服务 ID
const ImageDownloaderServiceID ServiceID = "ImageDownloader"

type IImageDownloaderService interface {
	AddConfig(sourceID string, config *config.ImageDownloaderConfig)
}

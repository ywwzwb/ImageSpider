package interfaces

import "ywwzwb/imagespider/models/config"

const ImageDownloaderDownloaderServiceID ServiceID = "ImageDownloader"

type IImageDownloaderService interface {
	AddConfig(sourceID string, config *config.ImageDownloaderConfig)
}

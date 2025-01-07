package plugins

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log/slog"
	"os"
	"path"
	"ywwzwb/imagespider/interfaces"
	"ywwzwb/imagespider/models/config"

	"github.com/strukturag/libheif/go/heif"
)

const ImageConvertPluginID string = "ImageConvert"

type ImageConvert struct {
	app    interfaces.IApplication
	config config.ImageConvertConfig
}

func newImageConverter() *ImageConvert {
	imageConverter := ImageConvert{}
	return &imageConverter
}

func init() {
	converter := newImageConverter()
	interfaces.Plugins[converter.ID()] = converter
}

func (i *ImageConvert) Name() string {
	return "ImageConvert"
}
func (i *ImageConvert) ID() string {
	return ImageConvertPluginID
}
func (i *ImageConvert) Load(app interfaces.IApplication) error {
	i.app = app
	i.config = app.GetAppConfig().ImageConvertConfig
	// 创建临时目录用于下载
	if err := os.MkdirAll(app.GetAppConfig().ImageDir, 0755); err != nil {
		slog.Error("create image dir failed", "path", app.GetAppConfig().ImageDir, "error", err)
		return err
	}
	return nil
}
func (i *ImageConvert) Unload() {
}
func (i *ImageConvert) GetService(serviceID interfaces.ServiceID) (interfaces.IService, error) {
	switch serviceID {
	case interfaces.ImageConvertServiceID:
		return i, nil
	}
	return nil, fmt.Errorf("service not found")
}
func (i *ImageConvert) Convert(input, hash string, finishCallback func(output string, err error)) {
	imageRelativeDirPath := path.Join(hash[0:2], hash[2:4], hash[4:6])
	imageFullDirPath := path.Join(i.app.GetAppConfig().ImageDir, imageRelativeDirPath)
	if err := os.MkdirAll(imageFullDirPath, 0755); err != nil {
		slog.Error("create image dir failed", "path", imageFullDirPath, "error", err)
		finishCallback("", err)
		return
	}
	imageRelativePath := path.Join(imageRelativeDirPath, hash+".heic")
	imageFullPath := path.Join(i.app.GetAppConfig().ImageDir, imageRelativePath)
	if _, err := os.Stat(imageFullPath); err == nil {
		finishCallback(imageRelativePath, nil)
		return
	}
	// Open the source file
	file, err := os.Open(input)
	if err != nil {
		slog.Error("failed to open file", "path", input, "err", err)
		finishCallback("", err)
		return
	}
	defer file.Close()
	file.Seek(0, 0)
	// Decode the image and get its format
	image, _, err := image.Decode(file)
	if err != nil {
		slog.Error("failed to decode image", "path", input, "err", err)
		finishCallback("", err)
		return
	}

	// Encode the image in HEIF format
	var losslessMode heif.LosslessMode
	if i.config.LosslessModeEnabled {
		losslessMode = heif.LosslessModeEnabled
	} else {
		losslessMode = heif.LosslessModeDisabled
	}
	ctx, err := heif.EncodeFromImage(image, heif.CompressionHEVC, i.config.Quality, losslessMode, heif.LoggingLevelNone)
	if err != nil {
		slog.Error("failed to encode image", "path", input, "err", err)
		finishCallback("", err)
		return
	}

	// Save the HEIF data to a file
	if err := ctx.WriteToFile(imageFullPath); err != nil {
		slog.Error("failed to write to file", "output", imageFullPath, "err", err)
		finishCallback("", err)
		return
	}
	finishCallback(imageRelativePath, nil)
}

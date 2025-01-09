package plugins

import (
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
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
func (i *ImageConvert) ConvertHEIC(input, output string) error {
	outputDir := path.Dir(output)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		slog.Error("create image output dir failed", "path", outputDir, "error", err)
		return err
	}
	if _, err := os.Stat(output); err == nil {
		slog.Info("image already exists", "path", output)
		return nil
	}
	// Open the source file
	file, err := os.Open(input)
	if err != nil {
		slog.Error("failed to open file", "path", input, "err", err)
		return err
	}
	defer file.Close()
	file.Seek(0, 0)
	// Decode the image and get its format
	image, _, err := image.Decode(file)
	if err != nil {
		slog.Error("failed to decode image", "path", input, "err", err)
		return err
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
		return err
	}

	// Save the HEIF data to a file
	if err := ctx.WriteToFile(output); err != nil {
		slog.Error("failed to write to file", "output", output, "err", err)
		return err
	}
	return nil
}
func (i *ImageConvert) ConvertPNG(input, output string) error {
	outputDir := path.Dir(output)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		slog.Error("create image output dir failed", "path", outputDir, "error", err)
		return err
	}
	if _, err := os.Stat(output); err == nil {
		slog.Info("image already exists", "path", output)
		return nil
	}
	// Open the source file
	file, err := os.Open(input)
	if err != nil {
		slog.Error("failed to open file", "path", input, "err", err)
		return err
	}
	defer file.Close()
	file.Seek(0, 0)
	// Decode the image and get its format
	image, _, err := image.Decode(file)
	if err != nil {
		slog.Error("failed to decode image", "path", input, "err", err)
		return err
	}
	outputFile, err := os.Create(output)
	if err == nil {
		slog.Error("failed to create file", "output", output, "err", err)
		return err
	}
	defer outputFile.Close()
	return png.Encode(outputFile, image)
}

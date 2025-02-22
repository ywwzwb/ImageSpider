package plugins

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"strings"
	"ywwzwb/imagespider/interfaces"
	"ywwzwb/imagespider/models/config"
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
	logger := slog.With("input", input, "output", output)
	outputDir := path.Dir(output)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		logger.Error("create image output dir failed", "path", outputDir, "error", err)
		return err
	}
	if _, err := os.Stat(output); err == nil {
		logger.Info("image already exists", "path", output)
		return nil
	}
	cmd := exec.Command("magick", input, output)
	var errOut strings.Builder
	cmd.Stderr = &errOut
	err := cmd.Run()
	if err != nil {
		logger.Error("convert image failed", "error", err, "stderr", errOut.String(), "exit code", cmd.ProcessState.ExitCode(), "cmd", cmd.String())
		return err
	}
	return nil
}

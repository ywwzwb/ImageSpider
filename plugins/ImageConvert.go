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
	return interfaces.ImageConvertPluginID
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

// GetDefaultOutputFormat 获取默认输出格式
func (i *ImageConvert) GetDefaultOutputFormat() string {
	if i.config.OutputFormat != "" {
		return i.config.OutputFormat
	}
	return "avif"
}

// GetDefaultQuality 获取默认质量
func (i *ImageConvert) GetDefaultQuality() int {
	if i.config.Quality > 0 {
		return i.config.Quality
	}
	return 85
}

// GetThumbnailConfigs 获取缩略图配置列表
func (i *ImageConvert) GetThumbnailConfigs() []config.ThumbnailConfig {
	return i.config.Thumbnails
}
func (i *ImageConvert) GetService(serviceID interfaces.ServiceID) (interfaces.IService, error) {
	switch serviceID {
	case interfaces.ImageConvertServiceID:
		return i, nil
	}
	return nil, fmt.Errorf("service not found")
}

// ConvertImage 转换图片格式，如果输出文件已存在则跳过
func (i *ImageConvert) ConvertImage(input, output string, quality int) error {
	logger := slog.With("input", input, "output", output, "quality", quality)

	// 确保输出目录存在
	outputDir := path.Dir(output)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		logger.Error("create output dir failed", "dir", outputDir, "error", err)
		return err
	}

	// 如果输出文件已存在，跳过转换
	if _, err := os.Stat(output); err == nil {
		logger.Debug("output file exists, skip conversion")
		return nil
	}

	// 构建 magick 命令
	args := []string{input}
	if quality > 0 {
		args = append(args, "-quality", fmt.Sprintf("%d", quality))
	}
	args = append(args, output)

	cmd := exec.Command("magick", args...)
	var errOut strings.Builder
	cmd.Stderr = &errOut

	if err := cmd.Run(); err != nil {
		logger.Error("convert image failed",
			"error", err,
			"stderr", errOut.String(),
			"exitCode", cmd.ProcessState.ExitCode(),
			"cmd", cmd.String())
		return err
	}

	logger.Info("image converted successfully")
	return nil
}

// GenerateThumbnail 生成缩略图，如果输出文件已存在则跳过
func (i *ImageConvert) GenerateThumbnail(input, output string, width, height int, options interfaces.ThumbnailOptions) error {
	logger := slog.With("input", input, "output", output, "width", width, "height", height)

	// 参数校验
	if width <= 0 && height <= 0 {
		return fmt.Errorf("width and height cannot both be zero")
	}

	// 确保输出目录存在
	outputDir := path.Dir(output)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		logger.Error("create thumbnail dir failed", "dir", outputDir, "error", err)
		return err
	}

	// 如果输出文件已存在，跳过生成
	if _, err := os.Stat(output); err == nil {
		logger.Debug("thumbnail exists, skip generation")
		return nil
	}

	// 构建 resize 参数
	resizeParam := ""
	if width > 0 && height > 0 {
		resizeParam = fmt.Sprintf("%dx%d>", width, height) // > 表示只缩小不放大
	} else if width > 0 {
		resizeParam = fmt.Sprintf("%dx", width)
	} else {
		resizeParam = fmt.Sprintf("x%d", height)
	}

	// 构建 magick 命令
	args := []string{input, "-resize", resizeParam}
	if options.Quality > 0 {
		args = append(args, "-quality", fmt.Sprintf("%d", options.Quality))
	}
	args = append(args, output)

	cmd := exec.Command("magick", args...)
	var errOut strings.Builder
	cmd.Stderr = &errOut

	if err := cmd.Run(); err != nil {
		logger.Error("generate thumbnail failed",
			"error", err,
			"stderr", errOut.String(),
			"exitCode", cmd.ProcessState.ExitCode(),
			"cmd", cmd.String())
		return err
	}

	logger.Info("thumbnail generated successfully")
	return nil
}

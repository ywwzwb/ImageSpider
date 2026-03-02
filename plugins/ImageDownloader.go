package plugins

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path"
	"sync/atomic"
	"time"
	"ywwzwb/imagespider/interfaces"
	"ywwzwb/imagespider/models"
	"ywwzwb/imagespider/models/config"
)

type ImageDownloader struct {
	app                 interfaces.IApplication
	configCount         atomic.Int32
	stopChain           chan bool
	stopFinishChain     chan bool
	downloadTempPath    string
	dbService           interfaces.IDBService
	imageConvertService interfaces.IImageConvertService
	goroutinCount       atomic.Int32
	defaultBatchSize    int
	defaultFetchInterval time.Duration
}

func newImageDownloader() *ImageDownloader {
	downloader := ImageDownloader{}
	downloader.stopChain = make(chan bool)
	downloader.stopFinishChain = make(chan bool)
	return &downloader
}

func init() {
	downloader := newImageDownloader()
	interfaces.Plugins[downloader.ID()] = downloader
}

func (i *ImageDownloader) Name() string {
	return "ImageDownloader"
}
func (i *ImageDownloader) ID() string {
	return interfaces.ImageDownloaderPluginID
}
func (i *ImageDownloader) Load(app interfaces.IApplication) error {
	i.app = app
	// 创建临时目录用于下载
	i.downloadTempPath = path.Join(app.GetAppConfig().WorkDir, "download_tmp")
	if err := os.MkdirAll(i.downloadTempPath, 0755); err != nil {
		slog.Error("create download temp dir failed", "path", i.downloadTempPath, "error", err)
		return err
	}
	// 获取数据库服务
	dbService, err := app.GetService(i.ID(), interfaces.DBPluginID, interfaces.DBServiceID)
	if err != nil {
		slog.Error("get db service failed", "error", err)
		return err
	}
	i.dbService = dbService.(interfaces.IDBService)
	imageConvertService, err := app.GetService(i.ID(), interfaces.ImageConvertPluginID, interfaces.ImageConvertServiceID)
	if err != nil {
		slog.Error("get image convert service failed", "error", err)
		return err
	}
	i.imageConvertService = imageConvertService.(interfaces.IImageConvertService)
	return nil
}
func (i *ImageDownloader) Unload() {
	for ; i.goroutinCount.Load() > 0; i.goroutinCount.Add(-1) {
		i.stopChain <- true
		<-i.stopFinishChain
	}
}
func (i *ImageDownloader) GetService(serviceID interfaces.ServiceID) (interfaces.IService, error) {
	switch serviceID {
	case interfaces.ImageDownloaderServiceID:
		return i, nil
	}
	return nil, fmt.Errorf("service not found")
}
func (i *ImageDownloader) AddConfig(sourceID string, cfg *config.ImageDownloaderConfig) {
	i.goroutinCount.Add(1)
	go i.downloadForSourceID(sourceID, cfg)
}
func (i *ImageDownloader) downloadForSourceID(sourceID string, cfg *config.ImageDownloaderConfig) {
	logger := slog.With("sourceID", sourceID)
	logger.Info("start download")

	// 使用配置的批次大小和间隔，如果没配置则使用默认值
	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = i.defaultBatchSize
	}
	fetchInterval := time.Duration(cfg.FetchInterval) * time.Second
	if fetchInterval <= 0 {
		fetchInterval = i.defaultFetchInterval
	}

	for {
		// 读取没有本地路径的资源
		metas := i.dbService.GetMetaLocalPathNULL(sourceID, batchSize)
		if len(metas) == 0 {
			logger.Info("no more data, check later")
			select {
			case <-i.stopChain:
				goto exit
			case <-time.After(fetchInterval):
				continue
			}
		}
		transport := &http.Transport{
			// 设置连接超时时间
			DialContext: (&net.Dialer{
				Timeout: time.Duration(cfg.ConnectTimeout) * time.Second,
			}).DialContext,
		}
		httpClient := &http.Client{
			Transport: transport,
		}
		for _, meta := range metas {
			select {
			case <-i.stopChain:
				goto exit
			default:
			}
			exit := false
			i.downloadImage(httpClient, sourceID, meta, cfg, &exit)
			if exit {
				goto exit
			}
		}
	}
exit:
	i.stopFinishChain <- true
	logger.Info("download routine exit")
}

func (i *ImageDownloader) downloadImage(httpClient *http.Client, sourceID string, meta models.ImageMeta, cfg *config.ImageDownloaderConfig, exit *bool) {
	hash := meta.Hash()
	logger := slog.With("sourceID", sourceID).With("metaID", meta.ID, "hash", hash)
	tempDownloadFilePath := path.Join(i.downloadTempPath, hash+path.Ext(meta.ImageURL))
	tempDownloadFilePathDownloading := tempDownloadFilePath + ".downloading"
	imageOutputPath := path.Join(hash[0:2], hash[2:4], hash[4:6], hash)
	imageOutputAbsolutePath := path.Join(i.app.GetAppConfig().ImageDir, imageOutputPath)

	// 使用配置的默认格式
	outputFormat := i.imageConvertService.GetDefaultOutputFormat()
	mainImagePath := imageOutputAbsolutePath + "." + outputFormat

	// 检查主图是否已存在
	if _, err := os.Stat(mainImagePath); err == nil {
		logger.Info("main image exists, save to database")
		i.saveImagePath(meta, imageOutputPath+"."+outputFormat)
		return
	}

	// 检查临时下载文件是否已存在
	if _, err := os.Stat(tempDownloadFilePath); err == nil {
		logger.Info("temp file exists, convert it")
		i.convertAndSave(tempDownloadFilePath, mainImagePath, imageOutputAbsolutePath, meta, logger)
		return
	}

	// 执行下载
	if !i.performDownload(httpClient, meta, cfg, tempDownloadFilePathDownloading, tempDownloadFilePath, logger, exit) {
		return
	}

	// 转换并保存
	i.convertAndSave(tempDownloadFilePath, mainImagePath, imageOutputAbsolutePath, meta, logger)
}

// performDownload 执行下载，返回是否成功
func (i *ImageDownloader) performDownload(httpClient *http.Client, meta models.ImageMeta, cfg *config.ImageDownloaderConfig,
	tempDownloadingPath, tempDownloadPath string, logger *slog.Logger, exit *bool) bool {

	// 尝试断点续传
	var startDownloadPos int64
	if stat, err := os.Stat(tempDownloadingPath); err == nil {
		startDownloadPos = stat.Size()
		logger.Info("try resume download from", "offset", startDownloadPos)
	}

	var downloadedSize int64 = startDownloadPos
	var expectedSize int64 = -1

	logger.Info("start download")
	var resp *http.Response
	var err error

	for idx := 0; idx < int(cfg.ErrorRetryMaxCount); idx++ {
		req, err := http.NewRequest("GET", meta.ImageURL, nil)
		if err != nil {
			logger.Error("create request failed", "error", err)
			return false
		}
		for k, v := range cfg.Headers {
			req.Header.Add(k, v)
		}
		if startDownloadPos > 0 {
			req.Header.Add("Range", fmt.Sprintf("bytes=%d-", startDownloadPos))
		}
		resp, err = httpClient.Do(req)
		if err != nil || (resp.StatusCode != 200 && resp.StatusCode != 206) {
			startDownloadPos = 0
			downloadedSize = 0
			os.Remove(tempDownloadingPath)
			select {
			case <-i.stopChain:
				*exit = true
				return false
			case <-time.After(time.Duration(cfg.ErrorRetryInterval) * time.Second):
				continue
			}
		}
		break
	}

	if resp == nil {
		logger.Error("fetch image failed", "error", err)
		i.markAsFailed(meta, logger)
		return false
	}
	defer resp.Body.Close()

	// 获取预期的内容长度
	if resp.StatusCode == 200 {
		expectedSize = resp.ContentLength
	} else if resp.StatusCode == 206 && startDownloadPos > 0 && resp.ContentLength > 0 {
		expectedSize = startDownloadPos + resp.ContentLength
	}

	// 保存到临时文件
	output, err := os.OpenFile(tempDownloadingPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		logger.Error("create temp file failed", "error", err)
		return false
	}

	for {
		select {
		case <-i.stopChain:
			*exit = true
			output.Close()
			return false
		default:
		}
		size, err := io.CopyN(output, resp.Body, 4*1024)
		downloadedSize += size
		if size == 0 || err != nil {
			if err != nil && err != io.EOF {
				logger.Error("download interrupted", "error", err, "downloadedSize", downloadedSize, "expectedSize", expectedSize)
			}
			break
		}
	}
	output.Close()

	// 验证文件大小
	if expectedSize > 0 && downloadedSize != expectedSize {
		logger.Error("download incomplete", "downloadedSize", downloadedSize, "expectedSize", expectedSize)
		os.Remove(tempDownloadingPath)
		return false
	}

	if downloadedSize == 0 {
		logger.Error("downloaded file is empty")
		os.Remove(tempDownloadingPath)
		return false
	}

	if err := os.Rename(tempDownloadingPath, tempDownloadPath); err != nil {
		logger.Error("rename temp file failed", "error", err)
		return false
	}

	logger.Info("download success", "size", downloadedSize, "expectedSize", expectedSize)
	return true
}

// convertAndSave 转换图片并保存到数据库
func (i *ImageDownloader) convertAndSave(tempPath, mainImagePath, imageOutputAbsolutePath string,
	meta models.ImageMeta, logger *slog.Logger) {
	// 使用配置的默认质量转换主图
	defaultQuality := i.imageConvertService.GetDefaultQuality()
	if err := i.imageConvertService.ConvertImage(tempPath, mainImagePath, defaultQuality); err != nil {
		logger.Error("convert main image failed", "error", err)
		i.markAsFailed(meta, logger)
		return
	}
	logger.Info("main image converted")

	// 根据配置生成缩略图
	thumbnailConfigs := i.imageConvertService.GetThumbnailConfigs()
	for _, tc := range thumbnailConfigs {
		thumbnailPath := imageOutputAbsolutePath + tc.Suffix + "." + tc.Format
		options := interfaces.ThumbnailOptions{Quality: tc.Quality}
		if err := i.imageConvertService.GenerateThumbnail(mainImagePath, thumbnailPath, tc.Width, tc.Height, options); err != nil {
			logger.Warn("generate thumbnail failed", "suffix", tc.Suffix, "error", err)
		} else {
			logger.Info("thumbnail generated", "suffix", tc.Suffix)
		}
	}

	// 保存到数据库，使用配置的格式
	outputFormat := i.imageConvertService.GetDefaultOutputFormat()
	hash := meta.Hash()
	imageOutputPath := path.Join(hash[0:2], hash[2:4], hash[4:6], hash) + "." + outputFormat
	i.saveImagePath(meta, imageOutputPath)
	os.Remove(tempPath)
}

// markAsFailed 标记为失败（保存空路径）
func (i *ImageDownloader) markAsFailed(meta models.ImageMeta, logger *slog.Logger) {
	empty := ""
	meta.LocalPath = &empty
	if err := i.dbService.UpdateLocalPathForMeta(meta); err != nil {
		logger.Error("update local path failed", "error", err)
	}
}

// saveImagePath 保存图片路径到数据库
func (i *ImageDownloader) saveImagePath(meta models.ImageMeta, imagePath string) {
	meta.LocalPath = &imagePath
	if err := i.dbService.UpdateLocalPathForMeta(meta); err != nil {
		slog.With("metaID", meta.ID).Error("update local path failed", "error", err)
	}
}

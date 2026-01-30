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

const ImageDownloaderPluginID string = "ImageDownloader"

const fetchBatchSize = 10
const fetchInterval = 60 * time.Second

type ImageDownloader struct {
	app                 interfaces.IApplication
	configCount         atomic.Int32
	stopChain           chan bool
	stopFinishChain     chan bool
	downloadTempPath    string
	dbService           interfaces.IDBService
	imageConvertService interfaces.IImageConvertService
	goroutinCount       atomic.Int32
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
	return ImageDownloaderPluginID
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
	dbService, err := app.GetService(i.ID(), DBPluginID, interfaces.DBServiceID)
	if err != nil {
		slog.Error("get db service failed", "error", err)
		return err
	}
	i.dbService = dbService.(interfaces.IDBService)
	imageConvertService, err := app.GetService(i.ID(), ImageConvertPluginID, interfaces.ImageConvertServiceID)
	if err != nil {
		slog.Error("get db service failed", "error", err)
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
	case interfaces.ImageDownloaderDownloaderServiceID:
		return i, nil
	}
	return nil, fmt.Errorf("service not found")
}
func (i *ImageDownloader) AddConfig(sourceID string, config *config.ImageDownloaderConfig) {
	i.goroutinCount.Add(1)
	go i.downloadForSourceID(sourceID, config)
}
func (i *ImageDownloader) downloadForSourceID(sourceID string, config *config.ImageDownloaderConfig) {
	logger := slog.With("sourceID", sourceID)
	logger.Info("start download")
	for {
		// 读取几条没有本地路径的资源
		metas := i.dbService.GetMetaLocalPathNULL(sourceID, fetchBatchSize)
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
				Timeout: time.Duration(config.ConnectTimeout) * time.Second,
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
			var exit bool = false
			i.downloadImage(httpClient, sourceID, meta, config, &exit)
			if exit {
				goto exit
			}
		}
	}
exit:
	i.stopFinishChain <- true
}
func (i *ImageDownloader) downloadImage(httpClient *http.Client, sourceID string, meta models.ImageMeta, config *config.ImageDownloaderConfig, exit *bool) {
	var req *http.Request
	var resp *http.Response = nil
	var output *os.File = nil
	var startDownloadPos int64 = 0
	var stat os.FileInfo
	var downloadedSize int64
	var expectedSize int64 = -1
	hash := meta.Hash()
	logger := slog.With("sourceID", sourceID).With("metaID", meta.ID, "hash", hash)
	tempDownloadFilePath := path.Join(i.downloadTempPath, hash+path.Ext(meta.ImageURL))
	tempDownloadFilePathDownloading := tempDownloadFilePath + ".downloading"
	imageOutputPath := path.Join(hash[0:2], hash[2:4], hash[4:6], hash)
	imageOutputAbsolutePath := path.Join(i.app.GetAppConfig().ImageDir, imageOutputPath)

	_, err := os.Stat(imageOutputAbsolutePath + ".avif")
	if err == nil {
		logger.Info("converted file exists, save it")
		goto save
	}
	_, err = os.Stat(tempDownloadFilePath)
	if err == nil {
		logger.Info("file download path exists, convert it")
		goto convert
	}
	stat, err = os.Stat(tempDownloadFilePathDownloading)
	if err == nil {
		startDownloadPos = stat.Size()
		downloadedSize = startDownloadPos
		logger.Info("try resume download from", "offset", startDownloadPos)
	}
	logger.Info("start download")
	for idx := 0; idx < int(config.ErrorRetryMaxCount); idx++ {
		req, err = http.NewRequest("GET", meta.ImageURL, nil)
		if err != nil {
			logger.Error("create request failed", "error", err)
			break
		}
		for k, v := range config.Headers {
			req.Header.Add(k, v)
		}
		if startDownloadPos > 0 {
			req.Header.Add("Range", fmt.Sprintf("bytes=%d-", startDownloadPos))
		}
		resp, err = httpClient.Do(req)
		if err != nil || (resp.StatusCode != 200 && resp.StatusCode != 206) {
			startDownloadPos = 0
			downloadedSize = 0
			os.Remove(tempDownloadFilePathDownloading)
			select {
			case <-i.stopChain:
				*exit = true
				return
			case <-time.After(time.Duration(config.ErrorRetryInterval) * time.Second):
				continue
			}
		}
		break
	}
	if resp == nil {
		logger.Error("fetch image failed, save empty path and skip for now", "error", err)
		empty := ""
		meta.LocalPath = &empty
		if err := i.dbService.UpdateLocalPathForMeta(meta); err != nil {
			logger.Error("update local path failed", "error", err)
			return
		}
		return
	}
	defer resp.Body.Close()

	// 获取预期的内容长度
	expectedSize = -1
	if resp.StatusCode == 200 {
		expectedSize = resp.ContentLength
	} else if resp.StatusCode == 206 && startDownloadPos > 0 {
		// 对于 Range 请求，计算总大小
		if resp.ContentLength > 0 {
			expectedSize = startDownloadPos + resp.ContentLength
		}
	}

	// 把resp.body 保存到 tempDownloadFilePath 中
	output, err = os.OpenFile(tempDownloadFilePathDownloading, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		logger.Error("create temp file failed", "error", err)
		return
	}

	for {
		select {
		case <-i.stopChain:
			*exit = true
			output.Close()
			return
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

	// 验证下载的文件大小是否符合预期
	if expectedSize > 0 && downloadedSize != expectedSize {
		logger.Error("download incomplete", "downloadedSize", downloadedSize, "expectedSize", expectedSize)
		// 删除不完整的文件
		os.Remove(tempDownloadFilePathDownloading)
		return
	}

	// 检查是否有下载错误
	if err != nil && err != io.EOF {
		logger.Error("write temp file failed", "error", err)
		os.Remove(tempDownloadFilePathDownloading)
		return
	}

	// 验证下载的文件不为空
	if downloadedSize == 0 {
		logger.Error("downloaded file is empty")
		os.Remove(tempDownloadFilePathDownloading)
		return
	}

	if err := os.Rename(tempDownloadFilePathDownloading, tempDownloadFilePath); err != nil {
		logger.Error("rename temp file failed", "error", err)
		return
	}
	logger.Info("download success", "size", downloadedSize, "expectedSize", expectedSize)
convert:
	err = i.imageConvertService.CompressImage(tempDownloadFilePath, imageOutputAbsolutePath+".avif")
	if err != nil {
		logger.Error("convert avif failed, save empty path and skip for now", "error", err)
		empty := ""
		meta.LocalPath = &empty
		if err := i.dbService.UpdateLocalPathForMeta(meta); err != nil {
			logger.Error("update local path failed", "error", err)
			return
		}
	}
	logger.Info("convert success, update local path")
save:
	_, err = os.Stat(imageOutputAbsolutePath + ".avif")
	if err != nil {
		logger.Info("image not exists, skip")
		return
	}
	imageOutputPath = imageOutputPath + ".avif"
	meta.LocalPath = &imageOutputPath
	if err := i.dbService.UpdateLocalPathForMeta(meta); err != nil {
		logger.Error("update local path failed", "error", err)
		return
	}
	os.Remove(tempDownloadFilePath)
}

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
}

func newImageDownloader() *ImageDownloader {
	downloader := ImageDownloader{}
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
	dbService, err := app.GetService(i.ID(), DBPluginID, interfaces.IDBServiceID)
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
	var idx int32
	for idx = 0; idx < i.configCount.Load(); idx++ {
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
	i.configCount.Add(1)
	go i.downloadForSourceID(sourceID, config)
}
func (i *ImageDownloader) downloadForSourceID(sourceID string, config *config.ImageDownloaderConfig) {
	logger := slog.With("sourceID", sourceID)
	logger.Info("start download")
	for {
		// 读取几条没有本地路径的资源
		metas := i.dbService.GetMetaWithoutLocalPath(sourceID, fetchBatchSize)
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
	logger := slog.With("sourceID", sourceID).With("metaID", meta.ID)
	hash := meta.Hash()
	tempDownloadFilePath := path.Join(i.downloadTempPath, hash+path.Ext(meta.ImageURL))
	tempDownloadFilePathDownloading := tempDownloadFilePath + ".downloading"
	imageOutputPath := path.Join(hash[0:2], hash[2:4], hash[4:6], hash+i.imageConvertService.GetFilextension())
	imageOutputAbsolutePath := path.Join(i.app.GetAppConfig().ImageDir, imageOutputPath)
	_, err := os.Stat(imageOutputAbsolutePath)
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
	defer resp.Body.Close()
	// 把resp.body 保存到 tempDownloadFilePath 中
	output, err = os.OpenFile(tempDownloadFilePathDownloading, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		logger.Error("create temp file failed", "error", err)
		return
	}

	_, err = io.Copy(output, resp.Body)
	output.Close()
	if err != nil {
		logger.Error("write temp file failed", "error", err)
		return
	}
	if err := os.Rename(tempDownloadFilePathDownloading, tempDownloadFilePath); err != nil {
		logger.Error("rename temp file failed", "error", err)
		return
	}
	logger.Info("download success, convert")
convert:
	err = i.imageConvertService.Convert(tempDownloadFilePath, imageOutputAbsolutePath)
	if err != nil {
		logger.Error("convert failed", "error", err)
		return
	}
	logger.Info("convert success, update local path")
save:
	meta.LocalPath = &imageOutputPath
	if err := i.dbService.UpdateLocalPathForMeta(meta); err != nil {
		logger.Error("update local path failed", "error", err)
		return
	}
	os.Remove(tempDownloadFilePath)
}

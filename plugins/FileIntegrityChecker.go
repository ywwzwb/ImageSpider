package plugins

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"ywwzwb/imagespider/interfaces"
	"ywwzwb/imagespider/models"
)

const FileIntegrityCheckerPluginID string = "FileIntegrityChecker"

const scanBatchSize = 100
const scanInterval = 60 * time.Second

type FileIntegrityChecker struct {
	app               interfaces.IApplication
	stopChain         chan bool
	stopFinishChain   chan bool
	dbService         interfaces.IDBService
	goroutinCount     atomic.Int32
}

func newFileIntegrityChecker() *FileIntegrityChecker {
	checker := FileIntegrityChecker{}
	checker.stopChain = make(chan bool)
	checker.stopFinishChain = make(chan bool)
	return &checker
}

func init() {
	checker := newFileIntegrityChecker()
	interfaces.Plugins[checker.ID()] = checker
}

func (f *FileIntegrityChecker) Name() string {
	return "FileIntegrityChecker"
}

func (f *FileIntegrityChecker) ID() string {
	return FileIntegrityCheckerPluginID
}

func (f *FileIntegrityChecker) Load(app interfaces.IApplication) error {
	f.app = app
	// 获取数据库服务
	dbService, err := app.GetService(f.ID(), DBPluginID, interfaces.DBServiceID)
	if err != nil {
		slog.Error("get db service failed", "error", err)
		return err
	}
	f.dbService = dbService.(interfaces.IDBService)
	return nil
}

func (f *FileIntegrityChecker) Unload() {
	for ; f.goroutinCount.Load() > 0; f.goroutinCount.Add(-1) {
		f.stopChain <- true
		<-f.stopFinishChain
	}
}

func (f *FileIntegrityChecker) GetService(serviceID interfaces.ServiceID) (interfaces.IService, error) {
	switch serviceID {
	case interfaces.FileIntegrityCheckerServiceID:
		return f, nil
	}
	return nil, fmt.Errorf("service not found")
}

func (f *FileIntegrityChecker) StartScanning(sourceID string) {
	f.goroutinCount.Add(1)
	go f.scanForSourceID(sourceID)
}

func (f *FileIntegrityChecker) scanForSourceID(sourceID string) {
	logger := slog.With("sourceID", sourceID)
	logger.Info("start file integrity scanning")
	for {
		// 获取需要扫描的图片
		offset := int64(0)
		for {
			select {
			case <-f.stopChain:
				goto exit
			default:
			}

			// 使用 ListDownloadedImageOfTags 获取已下载的图片
			result, err := f.dbService.ListDownloadedImageOfTags(sourceID, nil, offset, scanBatchSize)
			if err != nil {
				logger.Error("failed to list downloaded images", "error", err)
				select {
				case <-f.stopChain:
					goto exit
				case <-time.After(scanInterval):
					continue
				}
			}

			if len(result.ImageList) == 0 {
				logger.Info("no more data, check later")
				select {
				case <-f.stopChain:
					goto exit
				case <-time.After(scanInterval):
					goto restart
				}
			}

			for _, meta := range result.ImageList {
				select {
				case <-f.stopChain:
					goto exit
				default:
				}

				if meta.LocalPath == nil || *meta.LocalPath == "" {
					continue
				}

				filePath := path.Join(f.app.GetAppConfig().ImageDir, *meta.LocalPath)
				f.checkAndRemoveCorruptedFile(filePath, meta, sourceID)
			}

			offset += int64(len(result.ImageList))

			// 如果已经扫描完所有数据，重新开始一轮扫描
			if offset >= int64(result.TotalCount) {
				logger.Info("finished scanning all images, restarting")
				select {
				case <-f.stopChain:
					goto exit
				case <-time.After(scanInterval):
					goto restart
				}
			}

			select {
			case <-f.stopChain:
				goto exit
			case <-time.After(time.Second):
				continue
			}
		}
	restart:
		offset = 0
	}
exit:
	f.stopFinishChain <- true
}

func (f *FileIntegrityChecker) checkAndRemoveCorruptedFile(filePath string, meta models.ImageMeta, sourceID string) {
	logger := slog.With("filePath", filePath, "metaID", meta.ID)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		logger.Info("file does not exist, skip")
		return
	}

	// 验证图片完整性
	if err := f.validateImage(filePath); err != nil {
		logger.Error("file is corrupted, removing", "error", err)

		// 删除破损的文件
		if err := os.Remove(filePath); err != nil {
			logger.Error("failed to remove corrupted file", "error", err)
			return
		}

		// 从数据库中删除记录（设置 local_path 为 nil 或空）
		empty := ""
		meta.LocalPath = &empty
		if err := f.dbService.UpdateLocalPathForMeta(meta); err != nil {
			logger.Error("failed to update database", "error", err)
		}

		logger.Info("corrupted file removed successfully")
	} else {
		logger.Debug("file is valid")
	}
}

// validateImage 通过ImageMagick命令行工具验证图片完整性
func (f *FileIntegrityChecker) validateImage(filePath string) error {
	// 使用ImageMagick的identify命令验证图片
	cmd := exec.Command("magick", "identify", filePath)
	
	// 执行命令并捕获输出
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		// 如果命令执行失败，说明图片可能已损坏
		return fmt.Errorf("failed to identify image with ImageMagick: %w, output: %s", err, string(output))
	}
	
	// 如果输出为空，也认为是无效图片
	if len(strings.TrimSpace(string(output))) == 0 {
		return fmt.Errorf("ImageMagick returned empty output for file: %s", filePath)
	}
	
	return nil
}

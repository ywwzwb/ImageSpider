package plugins

import (
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"path"
	"strconv"
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
	for { f.goroutinCount.Load() > 0; f.goroutinCount.Add(-1) } {
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
	cmd := exec.Command("magick", "identify", "-regard-warnings", filePath)
	
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
	
	// 检查图片是否是纯色图片（可能是损坏图片的标志）
	isSolidColor, err := f.isSolidColorImage(filePath)
	if err != nil {
		return fmt.Errorf("error checking solid color: %w", err)
	}
	
	if isSolidColor {
		return fmt.Errorf("image appears to be solid color, likely corrupted: %s", filePath)
	}
	
	// 检查图片熵值（复杂度），过低的熵值可能表示图片质量差或损坏
	entropy, err := f.calculateEntropy(filePath)
	if err != nil {
		return fmt.Errorf("error calculating entropy: %w", err)
	}
	
	// 如果熵值太低（阈值设为2.0，可根据需要调整），认为图片有问题
	if entropy < 2.0 {
		return fmt.Errorf("image entropy too low (%f), likely corrupted: %s", entropy, filePath)
	}
	
	return nil
}

// isSolidColorImage 检查图片是否为纯色图片
func (f *FileIntegrityChecker) isSolidColorImage(filePath string) (bool, error) {
	// 使用ImageMagick计算图片的标准偏差，如果标准偏差为0则为纯色图片
	cmd := exec.Command("magick", filePath, "-format", "%[fx:standard_deviation]", "info:")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	
	sdStr := strings.TrimSpace(string(output))
	sd, err := strconv.ParseFloat(sdStr, 64)
	if err != nil {
		return false, fmt.Errorf("error parsing standard deviation: %w", err)
	}
	
	// 如果标准偏差接近0，说明是纯色图片
	return sd < 0.001, nil
}

// calculateEntropy 计算图片的信息熵，用于评估图片复杂度
func (f *FileIntegrityChecker) calculateEntropy(filePath string) (float64, error) {
	// 使用ImageMagick生成直方图并计算熵
	cmd := exec.Command("magick", filePath, "-define", "histogram:unique-colors=true", "histogram:info:-")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	
	// 分析直方图数据计算熵
	histogramLines := strings.Split(string(output), "\n")
	totalPixels := 0
	colorCounts := make(map[string]int)
	
	for _, line := range histogramLines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				countStr := strings.TrimSpace(parts[0])
				// 移除颜色计数中的数字和括号
				var count int
				fmt.Sscanf(countStr, "#%d", &count)
				totalPixels += count
				colorCounts[line] = count
			}
		}
	}
	
	if totalPixels == 0 {
		return 0, fmt.Errorf("could not determine pixel count")
	}
	
	// 计算熵值 H = -sum(p_i * log2(p_i))
	entropy := 0.0
	for _, count := range colorCounts {
		if count > 0 {
			prob := float64(count) / float64(totalPixels)
			if prob > 0 {
				entropy -= prob * logBase2(prob)
			}
		}
	}
	
	return entropy, nil
}

// logBase2 计算以2为底的对数
func logBase2(x float64) float64 {
	return 1.442695040888963 * math.Log(x) // 1/ln(2) ≈ 1.442695
}
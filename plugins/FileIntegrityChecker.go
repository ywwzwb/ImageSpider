package plugins

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
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
	app             interfaces.IApplication
	stopChain       chan bool
	stopFinishChain chan bool
	dbService       interfaces.IDBService
	goroutinCount   atomic.Int32
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
		select {
		case <-f.stopChain:
			goto exit
		default:
		}

		// 使用 ListDownloadedImagesWithUnknownStatus 获取未检测过的已下载图片
		result, err := f.dbService.ListDownloadedImagesWithUnknownStatus(sourceID, scanBatchSize)
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
			logger.Info("no more unverified images, waiting before next check")
			select {
			case <-f.stopChain:
				goto exit
			case <-time.After(scanInterval):
				continue
			}
		}

		logger.Info("scanning batch of images", "batch_size", len(result.ImageList), "remaining", result.TotalCount)

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

		// 扫描完一批后，继续获取下一批（状态已更新，不会重复扫描）
		logger.Debug("batch completed, fetching next batch")
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
		logger.Error("file is corrupted, marking as bad", "error", err)

		// 更新图片完整性状态为 bad
		if err := f.dbService.UpdateImageIntegrityStatus(sourceID, meta.ID, models.ImageIntegrityBad); err != nil {
			logger.Error("failed to update integrity status", "error", err)
			return
		}

		logger.Info("corrupted file marked as bad")
	} else {
		logger.Debug("file is valid")

		// 更新图片完整性状态为 good
		if err := f.dbService.UpdateImageIntegrityStatus(sourceID, meta.ID, models.ImageIntegrityGood); err != nil {
			logger.Error("failed to update integrity status", "error", err)
			return
		}

		logger.Debug("file marked as good")
	}
}

// validateImage 通过尝试解码图片来验证其完整性
func (f *FileIntegrityChecker) validateImage(filePath string) error {
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// 获取文件信息
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// 检查文件是否为空
	if stat.Size() == 0 {
		return fmt.Errorf("file is empty")
	}

	// 根据文件扩展名选择验证方式
	ext := path.Ext(filePath)
	if ext == ".heic" || ext == ".HEIC" {
		// 对于HEIC格式，使用magick验证
		return f.validateHEICImage(filePath)
	}

	// 对于其他格式，尝试解码图片
	_, _, err = image.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	return nil
}

// RunTest 运行破损检测测试
func (f *FileIntegrityChecker) RunTest() {
	// 测试图片配置，使用相对路径或环境变量
	testDir := os.Getenv("TEST_IMAGE_DIR")
	if testDir == "" {
		// 默认使用当前目录下的test_images文件夹
		testDir = "./test_images"
	}

	testImages := map[string]string{
		"bad_big.heic":    "大面积破损",
		"bad_small.heic":  "中等面积破损",
		"bad_little.heic": "小面积破损",
		"good_gray.heic":  "正常图片, 纯色背景",
		"good.heic":       "正常纹理图",
	}

	fmt.Println("=== File Integrity Checker Test ===")
	fmt.Printf("Test directory: %s\n\n", testDir)

	allPassed := true
	for filename, description := range testImages {
		filePath := path.Join(testDir, filename)
		err := f.validateHEICImage(filePath)

		// 判断期望结果：bad_开头的应该检测到破损（err != nil），good_开头的应该正常（err == nil）
		expectedCorrupted := strings.HasPrefix(filename, "bad_")
		actualCorrupted := err != nil

		status := "✓ PASS"
		if expectedCorrupted != actualCorrupted {
			status = "✗ FAIL"
			allPassed = false
		}

		result := "normal"
		if err != nil {
			result = fmt.Sprintf("CORRUPTED: %v", err)
		}

		fmt.Printf("[%s] %s (%s)\n", status, filename, description)
		fmt.Printf("      Result: %s\n\n", result)
	}

	if allPassed {
		fmt.Println("=== All tests passed! ===")
	} else {
		fmt.Println("=== Some tests failed! ===")
	}
}

// validateHEICImage 使用magick验证HEIC图片的完整性
func (f *FileIntegrityChecker) validateHEICImage(filePath string) error {
	// 使用magick identify验证图片完整性
	cmd := exec.Command("magick", "identify", "-format", "%w %h", filePath)
	var errOut strings.Builder
	var out strings.Builder
	cmd.Stderr = &errOut
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("magick identify failed: %w, stderr: %s", err, errOut.String())
	}

	// 获取图片尺寸
	output := strings.TrimSpace(out.String())
	if output == "" {
		return fmt.Errorf("failed to get image dimensions")
	}

	var width, height int
	_, parseErr := fmt.Sscanf(output, "%d %d", &width, &height)
	if parseErr != nil {
		return fmt.Errorf("failed to parse image dimensions: %w", parseErr)
	}

	// 检查图片尺寸是否合理（排除极小或无效的图片）
	if width < 10 || height < 10 {
		return fmt.Errorf("image dimensions too small: %dx%d", width, height)
	}

	// 检查是否有大量纯色区域（可能表示图片损坏或下半部分为灰色块）
	return f.checkForSolidColorRegion(filePath, width, height)
}

// analyzeRegion 分析图片区域的特征，返回是否纯色、标准差、颜色数
func (f *FileIntegrityChecker) analyzeRegion(filePath string, x, y, w, h int, regionName string) (bool, float64, int, error) {
	// 获取唯一颜色数量
	uniqueColors, err := f.getRegionUniqueColors(filePath, x, y, w, h)
	if err != nil {
		return false, 0, 0, fmt.Errorf("failed to get unique colors for %s region: %w", regionName, err)
	}

	// 检查是否完美纯色
	isSolid, stdDev, err := f.checkIfRegionIsPerfectlySolid(filePath, x, y, w, h)
	if err != nil {
		return false, 0, 0, fmt.Errorf("failed to check solid color for %s region: %w", regionName, err)
	}

	slog.Debug("region analysis",
		"region", regionName,
		"unique_colors", uniqueColors,
		"is_solid", isSolid,
		"std_dev", stdDev)

	return isSolid, stdDev, uniqueColors, nil
}

// checkForSolidColorRegion 检查图片是否存在大面积的纯色区域（可能表示损坏）
func (f *FileIntegrityChecker) checkForSolidColorRegion(filePath string, width, height int) error {
	// 如果图片高度太小，跳过检查
	if height < 100 {
		return nil
	}

	// 首先进行整体区域的对比检查
	if err := f.checkRegionDifferences(filePath, width, height); err != nil {
		// 如果检测到损坏（有错误返回），直接返回错误
		return err
	}

	// 然后检查底部纯色区域的大小占比
	// 如果底部纯色区域超过图片高度的50%，很可能是损坏
	return f.checkBottomSolidPercentage(filePath, width, height)
}

// checkBottomSolidPercentage 检查底部纯色区域占图片高度的百分比
func (f *FileIntegrityChecker) checkBottomSolidPercentage(filePath string, width, height int) error {
	// 从底部开始向上扫描，找出纯色区域的边界
	slog.Debug("checking bottom solid percentage",
		"file", path.Base(filePath))

	// 优化的扫描策略：从底部开始，快速跳过明显非纯色的区域
	const step = 100            // 增加步长到100px，减少扫描次数
	const minRegionHeight = 100 // 最小检查区域

	// 首先快速检查底部常见破损高度（10%-30%）
	testHeights := []float64{0.1, 0.15, 0.2, 0.25, 0.3}
	for _, ratio := range testHeights {
		regionHeight := int(float64(height) * ratio)
		if regionHeight < minRegionHeight {
			continue
		}

		startY := height - regionHeight
		isSolid, stdDev, colors, err := f.analyzeRegion(filePath, 0, startY, width, regionHeight, fmt.Sprintf("bottom_%.0f%%", ratio*100))
		if err != nil {
			slog.Warn("failed to analyze bottom region", "ratio", ratio, "error", err)
			continue
		}

		percentage := float64(regionHeight) / float64(height) * 100

		slog.Debug("quick bottom check",
			"ratio", ratio,
			"height", regionHeight,
			"percentage", percentage,
			"is_solid", isSolid,
			"std_dev", stdDev,
			"colors", colors)

		// 如果该区域是纯色
		if isSolid && stdDev < 0.01 && colors <= 10 {
			// 纯色区域超过5%就可能是损坏（进一步降低阈值）
			if percentage > 5 {
				// 检查纯色区域上方是否有突变
				checkHeight := 200
				if percentage < 15 {
					// 对于小面积破损(<15%)，检查更大的上方区域，避免漏检
					checkHeight = 400
				}
				if startY-checkHeight >= 0 {
					// 向上检查200px或400px区域，更好的检测小面积但深度的破损
					_, aboveStdDev, aboveColors, _ := f.analyzeRegion(filePath, 0, startY-checkHeight, width, checkHeight, "above_solid")

					// 智能阈值：根据上方颜色数量和std_dev综合判断
					// 真正的破损：上方颜色极多(>10000)且std_dev较大(>0.15)
					// 正常设计：上方颜色中等(1000-10000)且std_dev中等(0.05-0.15)
					// 将阈值从0.05提高到0.07，避免对正常图片的误判
					if aboveColors > 100 && aboveStdDev > 0.07 {
						// 上方颜色丰富，底部有大片纯色，很可能是损坏
						return fmt.Errorf("bottom %.0f%% solid, but above has many colors (%d) and variation (%.4f), likely corrupted",
							percentage, aboveColors, aboveStdDev)
					}
				}

				// 纯色区域超过70%，很可能是损坏
				if percentage > 70 {
					return fmt.Errorf("bottom %.0f%% of image is perfectly solid (std dev: %.4f), likely corrupted",
						percentage, stdDev)
				}

				// 纯色区域在5%-70%之间，上方没有突变，正常设计
				slog.Debug("solid region detected but no sharp transition above, likely normal design",
					"percentage", percentage, "colors", colors, "std_dev", stdDev)
				return nil
			}
		}
	}

	// 如果快速检查未发现问题，进行更细致的扫描（仅在需要时）
	// 只对小型图片或需要详细检查的图片执行此操作
	if width < 2000 || height < 2000 {
		slog.Debug("performing detailed scan for smaller image")
		for startY := height - step; startY >= 0; startY -= step {
			regionHeight := height - startY

			if regionHeight < minRegionHeight {
				continue
			}

			isSolid, stdDev, colors, err := f.analyzeRegion(filePath, 0, startY, width, regionHeight, "bottom_region")
			if err != nil {
				slog.Warn("failed to analyze bottom region", "startY", startY, "error", err)
				continue
			}

			percentage := float64(regionHeight) / float64(height) * 100

			// 如果在细致扫描中发现大片纯色区域
			if isSolid && stdDev < 0.01 && colors <= 10 && percentage > 10 {
				if startY-100 >= 0 {
					_, aboveStdDev, aboveColors, _ := f.analyzeRegion(filePath, 0, startY-100, width, 100, "above_solid")
					if aboveColors > 100 && aboveStdDev > 0.1 {
						return fmt.Errorf("bottom %.0f%% solid, but above has many colors (%d) and variation (%.4f), likely corrupted",
							percentage, aboveColors, aboveStdDev)
					}
				}
			}
		}
	}

	// 没有发现大片纯色区域
	slog.Debug("no large solid region detected at bottom")
	return nil
}

// checkRegionDifferences 检查顶部和底部区域的差异，如果检测到损坏则返回错误
func (f *FileIntegrityChecker) checkRegionDifferences(filePath string, width, height int) error {
	// 检查顶部20%和底部30%区域，对比差异
	topHeight := height * 2 / 10
	bottomStart := height * 7 / 10
	bottomHeight := height - bottomStart

	slog.Debug("checking region differences",
		"file", path.Base(filePath),
		"top_region", fmt.Sprintf("0,0 %dx%d", width, topHeight),
		"bottom_region", fmt.Sprintf("0,%d %dx%d", bottomStart, width, bottomHeight))

	// 获取顶部区域的统计信息
	topIsSolid, topStdDev, topUniqueColors, err := f.analyzeRegion(filePath, 0, 0, width, topHeight, "top")
	if err != nil {
		return err
	}

	// 获取底部区域的统计信息
	bottomIsSolid, bottomStdDev, bottomUniqueColors, err := f.analyzeRegion(filePath, 0, bottomStart, width, bottomHeight, "bottom")
	if err != nil {
		return err
	}

	// **核心判定逻辑：对比上下区域的差异**

	// 情况1：底部颜色很少（<=5），顶部颜色多很多（至少5倍）→ 损坏
	// 或者颜色差异超过3倍，且底部颜色很少（<=10），可能是小面积破损
	if bottomUniqueColors <= 5 && topUniqueColors > bottomUniqueColors*5 {
		return fmt.Errorf("bottom has very few colors (%d) vs top (%d), likely corrupted",
			bottomUniqueColors, topUniqueColors)
	}
	if bottomUniqueColors <= 10 && topUniqueColors > bottomUniqueColors*3 {
		slog.Debug("significant color difference detected (3x), checking middle region for confirmation",
			"bottom_colors", bottomUniqueColors, "top_colors", topUniqueColors)
		// 增加中间区域检查，避免误判
		middleStart := height * 4 / 10
		middleHeight := height * 2 / 10
		_, middleStdDev, middleColors, middleErr := f.analyzeRegion(filePath, 0, middleStart, width, middleHeight, "middle")
		if middleErr == nil && middleColors <= 10 && middleStdDev < 0.05 {
			slog.Debug("middle region also has few colors, likely normal design",
				"middle_colors", middleColors)
			return nil
		}
		return fmt.Errorf("significant color difference: bottom (%d) vs top (%d) colors, likely corrupted",
			bottomUniqueColors, topUniqueColors)
	}

	// 情况2：底部是纯色，顶部有更多变化 → 损坏
	if bottomIsSolid && bottomStdDev < 0.05 && topStdDev > bottomStdDev*2 {
		return fmt.Errorf("bottom is solid (%.4f, %d colors) but top has variation (%.4f), likely corrupted",
			bottomStdDev, bottomUniqueColors, topStdDev)
	}

	// 情况3：顶部和底部都是纯色 → 正常CG设计
	if topIsSolid && bottomIsSolid {
		slog.Debug("both regions are solid, normal design",
			"top_colors", topUniqueColors, "bottom_colors", bottomUniqueColors)
		return nil // 不返回错误
	}

	// 情况4：顶部和底部都有较多颜色 → 正常
	if topUniqueColors > 50 && bottomUniqueColors > 50 {
		slog.Debug("both regions have many colors, normal image",
			"top_colors", topUniqueColors, "bottom_colors", bottomUniqueColors)
		return nil // 不返回错误
	}

	// 情况5：添加中间区域检查，检测中部破损
	// 如果顶部和底部颜色都很多，但中部颜色很少，可能是中部破损
	middleStart := height * 4 / 10
	middleHeight := height * 2 / 10
	_, _, middleUniqueColors, middleErr := f.analyzeRegion(filePath, 0, middleStart, width, middleHeight, "middle")
	if middleErr == nil && middleUniqueColors <= 5 && topUniqueColors > middleUniqueColors*5 && bottomUniqueColors > middleUniqueColors*5 {
		return fmt.Errorf("middle region has very few colors (%d) vs top (%d) and bottom (%d), likely corrupted",
			middleUniqueColors, topUniqueColors, bottomUniqueColors)
	}

	// 其他情况，不判定为损坏（避免误判）
	slog.Debug("no corruption detected by comparison",
		"top_std_dev", topStdDev, "bottom_std_dev", bottomStdDev,
		"top_colors", topUniqueColors, "bottom_colors", bottomUniqueColors)
	return nil // 不返回错误
}

// getRegionUniqueColors 获取指定区域的唯一颜色数量
func (f *FileIntegrityChecker) getRegionUniqueColors(filePath string, x, y, w, h int) (int, error) {
	args := []string{
		filePath,
		"-crop", fmt.Sprintf("%dx%d+%d+%d", w, h, x, y),
		"-format", "%k", // %k 返回唯一颜色的数量
		"info:",
	}

	cmd := exec.Command("magick", args...)
	var out strings.Builder
	var errOut strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	err := cmd.Run()
	if err != nil {
		return 0, fmt.Errorf("magick command failed: %w, stderr: %s", err, errOut.String())
	}

	output := strings.TrimSpace(out.String())
	var uniqueColors int
	_, parseErr := fmt.Sscanf(output, "%d", &uniqueColors)
	if parseErr != nil {
		return 0, fmt.Errorf("failed to parse color count: %w", parseErr)
	}

	return uniqueColors, nil
}

// checkIfRegionIsPerfectlySolid 检查区域是否是完美纯色（标准差接近0）
func (f *FileIntegrityChecker) checkIfRegionIsPerfectlySolid(filePath string, x, y, w, h int) (bool, float64, error) {
	// 将区域转换为灰度并计算标准差
	// 如果标准差接近0，说明是完美纯色
	args := []string{
		filePath,
		"-crop", fmt.Sprintf("%dx%d+%d+%d", w, h, x, y),
		"-colorspace", "Gray",
		"-format", "%[fx:standard_deviation]",
		"info:",
	}

	cmd := exec.Command("magick", args...)
	var out strings.Builder
	var errOut strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	err := cmd.Run()
	if err != nil {
		return false, 0, fmt.Errorf("magick command failed: %w, stderr: %s", err, errOut.String())
	}

	output := strings.TrimSpace(out.String())
	var stdDev float64
	_, parseErr := fmt.Sscanf(output, "%f", &stdDev)
	if parseErr != nil {
		return false, 0, fmt.Errorf("failed to parse standard deviation: %w", parseErr)
	}

	// 如果标准差小于某个阈值，认为是纯色
	// CG图片即使有细节，stdDev也可能很小（0.15左右）
	// 完美纯色应该stdDev < 0.05
	isSolid := stdDev < 0.05

	return isSolid, stdDev, nil
}

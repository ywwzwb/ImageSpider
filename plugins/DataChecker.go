package plugins

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"sync/atomic"
	"time"
	"ywwzwb/imagespider/interfaces"
	"ywwzwb/imagespider/models"
)

const DataCheckerPluginID string = "DataChecker"

type DataChecker struct {
	app             interfaces.IApplication
	stopChain       chan bool
	stopFinishChain chan bool
	dbService       interfaces.IDBService
	goroutinCount   atomic.Int32
}

func newDataChecker() *DataChecker {
	checker := DataChecker{}
	checker.stopChain = make(chan bool)
	checker.stopFinishChain = make(chan bool)
	return &checker
}

func init() {
	checker := newDataChecker()
	interfaces.Plugins[checker.ID()] = checker
}

func (d *DataChecker) Name() string {
	return "DataChecker"
}

func (d *DataChecker) ID() string {
	return DataCheckerPluginID
}

func (d *DataChecker) Load(app interfaces.IApplication) error {
	d.app = app
	// 获取数据库服务
	dbService, err := app.GetService(d.ID(), DBPluginID, interfaces.DBServiceID)
	if err != nil {
		slog.Error("get db service failed", "error", err)
		return err
	}
	d.dbService = dbService.(interfaces.IDBService)
	return nil
}

func (d *DataChecker) Unload() {
	for ; d.goroutinCount.Load() > 0; d.goroutinCount.Add(-1) {
		d.stopChain <- true
		<-d.stopFinishChain
	}
}

func (d *DataChecker) GetService(serviceID interfaces.ServiceID) (interfaces.IService, error) {
	switch serviceID {
	case interfaces.DataCheckerServiceID:
		return d, nil
	}
	return nil, fmt.Errorf("service not found")
}

func (d *DataChecker) StartChecking(sourceID string) {
	d.goroutinCount.Add(1)
	go d.checkData(sourceID)
}

func (d *DataChecker) checkData(sourceID string) {
	for {
		select {
		case <-d.stopChain:
			goto exit
		default:
		}
		offset := 0
		hasBadMeta := false
		for {
			metas, err := d.dbService.ListDownloadedImageOfTags(sourceID, nil, int64(offset), int64(d.app.GetAppConfig().DataCheckerConfig.BatchSize))
			if err != nil {
				slog.Error("list downloaded image failed", "error", err)
				break
			}
			for _, meta := range metas.ImageList {
				path := path.Join(d.app.GetAppConfig().ImageDir, *meta.LocalPath)
				if _, err := os.Stat(path); err != nil {
					hasBadMeta = true
					slog.Error("image not found", "id", meta.ID, "path", path, "error", err)
					d.dbService.UpdateLocalPathForMeta(models.ImageMeta{
						ID:        meta.ID,
						LocalPath: nil,
					})
				}
			}
			if len(metas.ImageList) < d.app.GetAppConfig().DataCheckerConfig.BatchSize {
				if !hasBadMeta {
					slog.Info("check finish")
					break
				} else {
					slog.Info("check finish, but not all data checked, restart")
					offset = 0
					hasBadMeta = false
				}
			} else {
				offset += len(metas.ImageList)
			}
			select {
			case <-d.stopChain:
				goto exit
			case <-time.After(time.Duration(d.app.GetAppConfig().DataCheckerConfig.Interval) * time.Second):
				continue
			}
		}
		select {
		case <-d.stopChain:
			goto exit
		case <-time.After(5 * 60 * 60 * time.Second):
			continue
		}
	}
exit:
	d.stopFinishChain <- true
}

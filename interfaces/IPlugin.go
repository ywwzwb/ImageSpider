package interfaces

// 插件 ID 常量
const (
	DBPluginID                    = "DB"
	SpiderPluginID                = "spider"
	ImageDownloaderPluginID       = "imageDownloader"
	DataCheckerPluginID           = "dataChecker"
	ImageConvertPluginID          = "imageConvert"
	APIPluginID                   = "API"
	FileIntegrityCheckerPluginID  = "fileIntegrityChecker"
)

var Plugins = make(map[string]IPlugin)

type IPlugin interface {
	Name() string
	ID() string
	Load(app IApplication) error
	Unload()
	GetService(serviceID ServiceID) (IService, error)
}

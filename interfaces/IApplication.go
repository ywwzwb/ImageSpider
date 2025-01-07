package interfaces

import (
	"ywwzwb/imagespider/models/config"
	"ywwzwb/imagespider/models/runtimeConfig"
)

type IApplication interface {
	Run() error

	GetAppConfig() *config.Config
	GetRumtimeConfig() *runtimeConfig.Config

	GetService(callerPluginID, targetPluginID string, serviceID ServiceID) (IService, error)
}

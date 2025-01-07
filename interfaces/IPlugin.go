package interfaces

var Plugins = make(map[string]IPlugin)

type IPlugin interface {
	Name() string
	ID() string
	Load(app IApplication) error
	Unload()
	GetService(serviceID ServiceID) (IService, error)
}

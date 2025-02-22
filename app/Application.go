package app

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"
	"ywwzwb/imagespider/interfaces"
	"ywwzwb/imagespider/models/config"
	"ywwzwb/imagespider/models/runtimeConfig"
	_ "ywwzwb/imagespider/plugins"
	"ywwzwb/imagespider/util"

	"gopkg.in/yaml.v2"
)

type pluginMeta struct {
	plugin   interfaces.IPlugin
	depends  map[string]interface{}
	children map[string]interface{}
}
type Application struct {
	appConfig     config.Config
	runtimeConfig *runtimeConfig.Config
	pluginsMutex  sync.Mutex
	plugins       map[string]*pluginMeta
}

// create a singleton Application
var application *Application

func GetApplication() *Application {
	if application == nil {
		application = &Application{}
		application.plugins = make(map[string]*pluginMeta)
	}
	return application
}
func (app *Application) Run() error {
	configPathFromEnv, _ := os.LookupEnv("CONFIG_PATH")
	var configPath string
	flag.StringVar(&configPath, "c", "", "config file path")
	flag.Parse()
	if len(configPath) == 0 {
		configPath = configPathFromEnv
	}
	if len(configPath) == 0 {
		fmt.Fprintln(os.Stderr, "config file path is empty")
		flag.Usage()
		os.Exit(1)
	}
	configReader, err := os.Open(configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open config file failed", "configPath", configPath, "error", err)
		os.Exit(1)
	}
	defer configReader.Close()
	if err = yaml.NewDecoder(configReader).Decode(&app.appConfig); err != nil {
		fmt.Fprintln(os.Stderr, "decode config file failed", "configPath", configPath, "error", err)
		os.Exit(1)
	}
	// init logger
	util.InitLogger(app.appConfig.Logger)
	// init runtime config
	app.runtimeConfig = runtimeConfig.NewConfigFromPath(path.Join(app.appConfig.WorkDir, "/runtimeConfig.yaml"))
	// register plugins
	app.loadPlugins()
	// wait for signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	signal := <-c
	slog.Info("receive signal", "signal", signal)
	// time.Sleep(3 * time.Second)
	app.shutdown()
	return nil
}
func (app *Application) loadPlugins() {
	for _, plugin := range app.appConfig.Plugins {
		slog.Info("start load plugin", "plugin", plugin)
		if _, err := app.loadPlugin(plugin); err != nil {
			slog.Error("load plugin failed", "plugin", plugin, "error", err)
		}
	}
}
func (app *Application) loadPlugin(id string) (*pluginMeta, error) {
	p, ok := interfaces.Plugins[id]
	if !ok {
		return nil, fmt.Errorf("plugin not found, id:%s", id)
	}
	slog.Info("load plugin", "plugin", id)
	plugin := &pluginMeta{plugin: p}
	plugin.depends = make(map[string]interface{})
	plugin.children = make(map[string]interface{})
	app.pluginsMutex.Lock()
	app.plugins[id] = plugin
	app.pluginsMutex.Unlock()
	err := p.Load(app)
	if err == nil {
		slog.Info("load plugin finish", "plugin", id)
		return plugin, nil
	}
	slog.Error("load plugin failed", "plugin", id, "error", err)
	p.Unload()
	app.pluginsMutex.Lock()
	delete(app.plugins, id)
	app.pluginsMutex.Unlock()

	return nil, err
}
func (app *Application) shutdown() {
	slog.Info("shutdown begin")
	for {
		var unloadPlugins = make(map[string]*pluginMeta)
		app.pluginsMutex.Lock()
		if len(app.plugins) == 0 {
			app.pluginsMutex.Unlock()
			slog.Info("all plugins are unloaded")
			break
		}
		for pluginID, pluginMeta := range app.plugins {
			// 移除没有子插件的插件
			if len(pluginMeta.children) == 0 {
				unloadPlugins[pluginID] = pluginMeta
				delete(app.plugins, pluginID)
			}
		}
		for pluginID, pluginMeta := range unloadPlugins {
			// 从依赖插件中, 将自己移除
			for dependID := range pluginMeta.depends {
				delete(app.plugins[dependID].children, pluginID)
			}
		}
		if len(unloadPlugins) == 0 && len(app.plugins) > 0 {
			slog.Error("plugin referer may has dead loop")
			app.pluginsMutex.Unlock()
			break
		}
		app.pluginsMutex.Unlock()
		for pluginID, pluginMeta := range unloadPlugins {
			slog.Info("unload plugin", "plugin", pluginID)
			pluginMeta.plugin.Unload()
			slog.Info("unload plugin finish", "plugin", pluginID)
		}
	}
	app.runtimeConfig.Save()
	slog.Info("shutdown finish")
}

func (app *Application) GetAppConfig() *config.Config {
	return &app.appConfig
}
func (app *Application) GetRuntimeConfig() *runtimeConfig.Config {
	return app.runtimeConfig
}
func (app *Application) GetService(callerPluginID, targetPluginID string, serviceID interfaces.ServiceID) (interfaces.IService, error) {
	var callerPluginPlugin *pluginMeta
	var targePluginPlugin *pluginMeta
	app.pluginsMutex.Lock()
	callerPluginPlugin, ok := app.plugins[callerPluginID]
	if !ok {
		return nil, fmt.Errorf("caller plugin not found, id:%s", callerPluginID)
	}
	targePluginPlugin, ok = app.plugins[targetPluginID]
	app.pluginsMutex.Unlock()
	if !ok {
		var err error
		targePluginPlugin, err = app.loadPlugin(targetPluginID)
		if err != nil {
			return nil, err
		}
	}
	service, err := targePluginPlugin.plugin.GetService(serviceID)
	if err != nil {
		return nil, err
	}
	app.pluginsMutex.Lock()
	callerPluginPlugin.depends[targetPluginID] = nil
	targePluginPlugin.children[callerPluginID] = nil
	app.pluginsMutex.Unlock()
	return service, nil
}

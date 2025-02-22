package runtimeConfig

import (
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"gopkg.in/yaml.v2"
)

const (
	runTimeConfigV1 string = "v1"
)

type Config struct {
	path                  string
	Version               string `json:"version" yaml:"version"`
	lastFetchPageStackMtx sync.RWMutex
	LastFetchPageStack    map[string][]int64 `json:"lastFetchPageStack" yaml:"lastFetchPageStack"`
	saveMtx               sync.Mutex
}

func NewConfigFromPath(path string) *Config {
	emptyConfig := &Config{}
	emptyConfig.Version = runTimeConfigV1
	emptyConfig.path = path
	emptyConfig.LastFetchPageStack = make(map[string][]int64)
	configReader, err := os.Open(path)
	if err != nil {
		slog.Warn("empty config file", "path", path)
		return emptyConfig
	}
	defer configReader.Close()
	currentConfig := &Config{}
	if err = yaml.NewDecoder(configReader).Decode(&currentConfig); err != nil {
		slog.Error("parse config file failed", "path", path, "error", err)
		return emptyConfig
	}
	if currentConfig.Version != runTimeConfigV1 {
		slog.Error("unsupporteded config version", "version", emptyConfig.Version)
		return emptyConfig
	}
	slog.Debug("current run config", "config", currentConfig)
	currentConfig.path = path
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	go func() {
		for {
			<-c
			currentConfig.Save()
		}
	}()
	return currentConfig
}
func (c *Config) Save() error {
	slog.Debug("save config", "path", c.path)
	c.saveMtx.Lock()
	defer c.saveMtx.Unlock()
	dirpath := filepath.Dir(c.path)
	if err := os.MkdirAll(dirpath, 0755); err != nil {
		slog.Error("create config file dir failed", "path", dirpath, "error", err)
		return err
	}
	configWriter, err := os.Create(c.path)
	if err != nil {
		slog.Error("open config file failed", "path", c.path, "error", err)
		return err
	}
	defer configWriter.Close()
	err = yaml.NewEncoder(configWriter).Encode(c)
	if err != nil {
		slog.Error("save config file failed", "path", c.path, "error", err)
		return err
	}
	slog.Debug("save config file finish", "path", c.path)
	return nil
}
func (c *Config) ReplaceStackTop(id string, newVal int64) *int64 {
	logger := slog.With("source id", id).With("new val", newVal)
	logger.Debug("replace stack top")
	var oldTop *int64 = nil
	for {
		c.lastFetchPageStackMtx.Lock()
		defer c.lastFetchPageStackMtx.Unlock()
		lastStack, ok := c.LastFetchPageStack[id]
		if !ok || len(lastStack) == 0 {
			logger.Debug("stack empty, insert")
			c.LastFetchPageStack[id] = []int64{newVal}
			oldTop = nil
			break
		}
		oldTop = &lastStack[len(lastStack)-1]
		lastStack[len(lastStack)-1] = newVal
		logger.Debug("replace finish", "old top", oldTop)
		break
	}
	c.Save()
	return oldTop
}
func (c *Config) AppendStack(id string, newVal int64) {
	logger := slog.With("source id", id).With("new val", newVal)
	logger.Debug("append stack")
	for {
		c.lastFetchPageStackMtx.Lock()
		defer c.lastFetchPageStackMtx.Unlock()
		lastStack, ok := c.LastFetchPageStack[id]
		if !ok || len(lastStack) == 0 {
			c.LastFetchPageStack[id] = []int64{newVal}
			logger.Debug("empty stack, insert to top")
			break
		}
		if lastStack[len(lastStack)-1] == newVal {
			logger.Debug("stack top is same, ignore")
			return
		}
		lastStack = append(lastStack, newVal)
		c.LastFetchPageStack[id] = lastStack
		logger.Debug("insert finish")
		break
	}
	c.Save()
}
func (c *Config) StackTop(id string) *int64 {
	logger := slog.With("source id", id)
	logger.Debug("get stack top")
	c.lastFetchPageStackMtx.RLock()
	defer c.lastFetchPageStackMtx.RUnlock()
	lastStack, ok := c.LastFetchPageStack[id]
	if !ok || len(lastStack) == 0 {
		logger.Debug("stack empty")
		return nil
	}
	top := &lastStack[len(lastStack)-1]
	logger.Debug("get stack top finish", "top", top)
	return top
}
func (c *Config) StackPop(id string) *int64 {
	logger := slog.With("source id", id)
	logger.Debug("pop stack")
	var oldTop *int64 = nil
	for {
		c.lastFetchPageStackMtx.Lock()
		defer c.lastFetchPageStackMtx.Unlock()
		lastStack, ok := c.LastFetchPageStack[id]
		if !ok || len(lastStack) == 0 {
			logger.Debug("stack empty")
			return nil
		}
		oldTop = &lastStack[len(lastStack)-1]
		c.LastFetchPageStack[id] = lastStack[:len(lastStack)-1]
		logger.Debug("pop finish", "old top", oldTop)
		break
	}
	c.Save()
	return oldTop

}

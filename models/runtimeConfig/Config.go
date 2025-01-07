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
		slog.Info("empty config file", "path", path)
		return emptyConfig
	}
	defer configReader.Close()
	currentConfig := &Config{}
	if err = yaml.NewDecoder(configReader).Decode(&currentConfig); err != nil {
		slog.Error("parse config file failed", "path", path, "error", err)
		return emptyConfig
	}
	if currentConfig.Version != runTimeConfigV1 {
		slog.Error("unsupported config version", "version", emptyConfig.Version)
		return emptyConfig
	}
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
	return yaml.NewEncoder(configWriter).Encode(c)
}
func (c *Config) ReplaceStackTop(id string, newVal int64) *int64 {
	var oldTop *int64 = nil
	for {
		c.lastFetchPageStackMtx.Lock()
		defer c.lastFetchPageStackMtx.Unlock()
		lastStack, ok := c.LastFetchPageStack[id]
		if !ok || len(lastStack) == 0 {
			c.LastFetchPageStack[id] = []int64{newVal}
			oldTop = nil
			break
		}
		oldTop = &lastStack[len(lastStack)-1]
		lastStack[len(lastStack)-1] = newVal
		break
	}
	c.Save()
	return oldTop
}
func (c *Config) AppndStack(id string, newVal int64) {
	for {
		c.lastFetchPageStackMtx.Lock()
		defer c.lastFetchPageStackMtx.Unlock()
		lastStack, ok := c.LastFetchPageStack[id]
		if !ok || len(lastStack) == 0 {
			c.LastFetchPageStack[id] = []int64{newVal}
			break
		}
		if lastStack[len(lastStack)-1] == newVal {
			return
		}
		lastStack = append(lastStack, newVal)
		c.LastFetchPageStack[id] = lastStack
		break
	}
	c.Save()
}
func (c *Config) StackTop(id string) *int64 {
	c.lastFetchPageStackMtx.RLock()
	defer c.lastFetchPageStackMtx.RUnlock()
	lastStack, ok := c.LastFetchPageStack[id]
	if !ok || len(lastStack) == 0 {
		return nil
	}
	return &lastStack[len(lastStack)-1]
}
func (c *Config) StackPop(id string) *int64 {
	var oldTop *int64 = nil
	for {
		c.lastFetchPageStackMtx.Lock()
		defer c.lastFetchPageStackMtx.Unlock()
		lastStack, ok := c.LastFetchPageStack[id]
		if !ok || len(lastStack) == 0 {
			return nil
		}
		oldTop = &lastStack[len(lastStack)-1]
		c.LastFetchPageStack[id] = lastStack[:len(lastStack)-1]
		break
	}
	c.Save()
	return oldTop

}

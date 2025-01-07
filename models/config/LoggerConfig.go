package config

import "log/slog"

type LogFileConfig struct {
	Path            string `json:"path" yaml:"path"`
	MaxLogFileCount int    `json:"maxLogFileCount" yaml:"maxLogFileCount"`
	MaxLogFileSize  int    `json:"maxLogFileSize" yaml:"maxLogFileSize"`
}

type LoggerConfig struct {
	Level   slog.Level    `json:"level" yaml:"level"`
	File    LogFileConfig `json:"file" yaml:"file"`
	Console bool          `json:"console" yaml:"console,omitempty"`
}

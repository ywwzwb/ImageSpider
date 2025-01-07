package config

type ImageDownloaderConfig struct {
	Headers            map[string]string `json:"headers" yaml:"headers"`
	ErrorRetryInterval uint              `json:"errorRetryInterval" yaml:"errorRetryInterval"` // in seconds
	ErrorRetryMaxCount uint              `json:"errorRetryMaxCount" yaml:"errorRetryMaxCount"`
	ConnectTimeout     int               `json:"connectTimeout" yaml:"connectTimeout"` // in seconds
}

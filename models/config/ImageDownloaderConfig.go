package config

type ImageDownloaderConfig struct {
	Headers            map[string]string `json:"headers" yaml:"headers"`
	ErrorRetryInterval uint              `json:"errorRetryInterval" yaml:"errorRetryInterval"` // in seconds
	ErrorRetryMaxCount uint              `json:"errorRetryMaxCount" yaml:"errorRetryMaxCount"`
	ConnectTimeout     int               `json:"connectTimeout" yaml:"connectTimeout"` // in seconds
	BatchSize          int               `json:"batchSize" yaml:"batchSize"`          // 每次获取的任务数量，默认 10
	FetchInterval      int               `json:"fetchInterval" yaml:"fetchInterval"`  // 获取间隔（秒），默认 60
}

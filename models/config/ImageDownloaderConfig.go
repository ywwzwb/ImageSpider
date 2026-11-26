package config

type ImageDownloaderConfig struct {
	Headers            map[string]string `json:"headers" yaml:"headers"`
	ErrorRetryInterval uint              `json:"errorRetryInterval" yaml:"errorRetryInterval"` // in seconds
	ErrorRetryMaxCount uint              `json:"errorRetryMaxCount" yaml:"errorRetryMaxCount"`
	ConnectTimeout     int               `json:"connectTimeout" yaml:"connectTimeout"` // in seconds
	BatchSize          int               `json:"batchSize" yaml:"batchSize"`          // 每次获取的任务数量，默认 10
	FetchInterval      int               `json:"fetchInterval" yaml:"fetchInterval"`  // 获取间隔（秒），默认 60
	// FailureRetryMaxCount 下载失败后最多尝试次数（累计失败次数达到该值后标记为永久失败），默认 5
	FailureRetryMaxCount uint `json:"failureRetryMaxCount" yaml:"failureRetryMaxCount"`
	// FailureRetryInterval 下载失败后再次尝试的间隔（秒），默认 3600
	FailureRetryInterval uint `json:"failureRetryInterval" yaml:"failureRetryInterval"`
}

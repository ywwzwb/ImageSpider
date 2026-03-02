package config

type FileIntegrityCheckerConfig struct {
	Interval      int `json:"interval" yaml:"interval"`           // 检查间隔（秒）
	BatchSize     int `json:"batchSize" yaml:"batchSize"`         // 每次扫描数量
	ScanInterval  int `json:"scanInterval" yaml:"scanInterval"`   // 扫描间隔（秒）
}

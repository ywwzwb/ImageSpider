package config

type DataCheckerConfig struct {
	Interval        int `json:"interval" yaml:"interval"`
	BatchSize       int `json:"batchSize" yaml:"batchSize"`
	RestartInterval int `json:"restartInterval" yaml:"restartInterval"`
}

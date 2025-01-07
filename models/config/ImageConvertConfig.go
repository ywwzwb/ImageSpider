package config

type ImageConvertConfig struct {
	Quality             int  `json:"quality" yaml:"quality"`
	LosslessModeEnabled bool `json:"losslessModeEnabled" yaml:"losslessModeEnabled"`
}

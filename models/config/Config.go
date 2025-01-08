package config

import "encoding/json"

type SpiderList map[string]*SpiderConfig
type Config struct {
	Spiders            SpiderList         `json:"spiders" yaml:"spiders"`
	ImageConvertConfig ImageConvertConfig `json:"imageConverter" yaml:"imageConverter"`
	Logger             LoggerConfig       `json:"logger" yaml:"logger"`
	ImageDir           string             `json:"imageDir" yaml:"imageDir"`
	WorkDir            string             `json:"workDir" yaml:"workDir"`
	DatabaseConfig     DatabaseConfig     `json:"database" yaml:"database"`
	Plugins            []string           `json:"plugins" yaml:"plugins"`
	APIConfig          APIConfig          `json:"api" yaml:"api"`
}

func (a *SpiderList) UnmmarshalJSON(data []byte) error {
	var s map[string]*SpiderConfig
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*a = s
	for id := range s {
		(*a)[id].ID = id
	}
	return nil
}

func (a *SpiderList) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s map[string]*SpiderConfig
	if err := unmarshal(&s); err != nil {
		return err
	}
	*a = s
	for id, _ := range s {
		(*a)[id].ID = id
	}
	return nil
}

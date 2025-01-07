package config

type MetaDownloaderConfig struct {
	ErrorRetryInterval             uint `json:"errorRetryInterval" yaml:"errorRetryInterval"` // in seconds
	ErrorRetryMaxCount             uint `json:"errorRetryMaxCount" yaml:"errorRetryMaxCount"`
	StateMachineErrorRetryInterval uint `json:"stateMachineErrorRetryInterval" yaml:"stateMachineErrorRetryInterval"` // in seconds
	RefreshInterval                uint `json:"refreshInterval" yaml:"refreshInterval"`                               // in seconds
	ConnectTimeout                 int  `json:"connectTimeout" yaml:"connectTimeout"`                                 // in seconds
}
type ListParser struct {
	URLTemplate string            `json:"urlTemplate" yaml:"urlTemplate"`
	Headers     map[string]string `json:"headers" yaml:"headers"`
	IDList      HTMLParserConfig  `json:"id" yaml:"id"`
	PageNum     HTMLParserConfig  `json:"pageNum" yaml:"pageNum"`
	NextPage    HTMLParserConfig  `json:"nextPage" yaml:"nextPage"`
}
type MetaParser struct {
	URLTemplate string             `json:"urlTemplate" yaml:"urlTemplate"`
	Headers     map[string]string  `json:"headers" yaml:"headers"`
	Tags        []HTMLParserConfig `json:"tags" yaml:"tags"`
	ImageURL    HTMLParserConfig   `json:"imageURL" yaml:"imageURL"`
	PostTime    HTMLParserConfig   `json:"postTime" yaml:"postTime"`
}

type SpiderConfig struct {
	ID                    string                `json:"id" yaml:"id"`
	Name                  string                `json:"name" yaml:"name"`
	MetaDownloaderConfig  MetaDownloaderConfig  `json:"metaDownloader" yaml:"metaDownloader"`
	ListParser            ListParser            `json:"listParser" yaml:"listParser"`
	MetaParser            MetaParser            `json:"metaParser" yaml:"metaParser"`
	ImageDownloaderConfig ImageDownloaderConfig `json:"imageDownloader" yaml:"imageDownloader"`
}

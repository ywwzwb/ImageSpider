package config

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
)

type AttributeType int

const (
	AttributeTypeInnerText AttributeType = iota
	AttributeTypeHref
	AttributeTypeTitle
)

func (a *AttributeType) fromString(s string) error {
	switch strings.ToLower(s) {
	case "innertext":
		*a = AttributeTypeInnerText
	case "href":
		*a = AttributeTypeHref
	case "title":
		*a = AttributeTypeTitle
	default:
		return errors.New("invalid attribute type: " + s)
	}
	return nil
}
func (a *AttributeType) UnmmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	return a.fromString(s)
}
func (a *AttributeType) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	return a.fromString(s)
}

type OperatorType int

const (
	OperatorContain OperatorType = iota
)

func (o *OperatorType) fromString(s string) error {
	switch strings.ToLower(s) {
	case "contains":
		*o = OperatorContain
	default:
		return errors.New("invalid attribute type: " + s)
	}
	return nil
}
func (o *OperatorType) UnmmarshalJSON(data []byte) error {
	// ignore case when unmarshal
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	return o.fromString(s)
}

func (o *OperatorType) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	return o.fromString(s)
}

type ReplacerConfig struct {
	Regex       *regexp.Regexp
	Replacement string
}
type ValueConfig struct {
	Attribute      AttributeType
	ReplacerConfig *ReplacerConfig
}

func (a *ValueConfig) fromObject(v interface{}) error {
	if s, ok := v.(string); ok {
		// simple string
		return a.Attribute.fromString(s)
	}
	m, ok := v.(map[any]any)
	if !ok {
		return errors.New("invalid value config")
	}
	attribute, ok := m["attribute"]
	if !ok {
		return errors.New("attribute is required")
	}
	attributeStr, ok := attribute.(string)
	if !ok {
		return errors.New("attribute must be a string")
	}
	if err := a.Attribute.fromString(attributeStr); err != nil {
		return err
	}
	replacer, ok := m["replacer"]
	if !ok {
		return nil
	}
	replacerMap, ok := replacer.(map[any]any)
	if !ok {
		return errors.New("replacer must be a map")
	}
	replaceConfig := &ReplacerConfig{}
	regex, ok := replacerMap["regex"]
	if !ok {
		return errors.New("regex is required")
	}
	regexStr, ok := regex.(string)
	if !ok {
		return errors.New("regex must be a string")
	}
	if regexExpression, err := regexp.Compile(regexStr); err != nil {
		return errors.New("invalid regex: " + regexStr + ", error" + err.Error())
	} else {
		replaceConfig.Regex = regexExpression
	}

	replacement, ok := replacerMap["replacement"]
	if !ok {
		return errors.New("replacement is required")
	}
	replacementStr, ok := replacement.(string)
	if !ok {
		return errors.New("replacement must be a string")
	}
	replaceConfig.Replacement = replacementStr
	a.ReplacerConfig = replaceConfig
	return nil
}
func (a *ValueConfig) UnmmarshalJSON(data []byte) error {
	// ignore case when unmarshal
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	return a.fromObject(v)
}
func (a *ValueConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// ignore case when unmarshal
	var v interface{}
	if err := unmarshal(&v); err != nil {
		return err
	}
	return a.fromObject(v)
}

type ElementMatherConfig struct {
	Attribute    AttributeType `json:"attribute" yaml:"attribute"`
	OperatorType OperatorType  `json:"operator" yaml:"operator"`
	Value        string        `json:"value" yaml:"value"`
}

type HTMLParserConfig struct {
	Selector            string               `json:"selector" yaml:"selector"`
	Value               ValueConfig          `json:"value" yaml:"value"`
	ElementMatherConfig *ElementMatherConfig `json:"matcher" yaml:"matcher"`
	Ext                 map[string]string    `json:"ext" yaml:"ext"`
}

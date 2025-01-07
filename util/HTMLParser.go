package util

import (
	"fmt"
	"strings"
	"ywwzwb/imagespider/models/config"

	"github.com/PuerkitoBio/goquery"
)

type HTMLParser struct {
	config *config.HTMLParserConfig
}

func NewParser(config *config.HTMLParserConfig) *HTMLParser {
	parser := &HTMLParser{}
	parser.config = config
	return parser
}
func (p *HTMLParser) Parse(doc *goquery.Document) ([]string, error) {
	result := make([]string, 0)
	elements := doc.Find(p.config.Selector)
	if elements == nil {
		return result, fmt.Errorf("no elements found, selector: %s", p.config.Selector)
	}
	elements.Each(func(i int, s *goquery.Selection) {
		if !p.match(s, p.config.ElementMatherConfig) {
			return
		}
		if value, ok := p.getValue(s, &p.config.Value); ok {
			result = append(result, value)
		}
	})
	return result, nil
}
func (p *HTMLParser) match(selecton *goquery.Selection, m *config.ElementMatherConfig) bool {
	if m == nil {
		return true
	}
	val, ok := p.getAttribute(selecton, m.Attribute)
	if !ok {
		return false
	}
	switch m.OperatorType {
	case config.OperatorContain:
		return strings.Contains(val, m.Value)
	default:
		return false
	}
}
func (p *HTMLParser) getAttribute(s *goquery.Selection, attr config.AttributeType) (string, bool) {
	switch attr {
	case config.AttributeTypeInnnerText:
		return s.Text(), true
	case config.AttributeTypeHref:
		return s.Attr("href")
	case config.AttributeTypeTitle:
		return s.Attr("title")
	default:
		break
	}
	return "", false
}
func (p *HTMLParser) getValue(s *goquery.Selection, valueConfig *config.ValueConfig) (string, bool) {
	value, ok := p.getAttribute(s, valueConfig.Attribute)
	if !ok {
		return "", false
	}
	if valueConfig.ReplacerConfig == nil {
		return value, true
	}
	return valueConfig.ReplacerConfig.Regex.ReplaceAllString(value, valueConfig.ReplacerConfig.Replacement), true
}

package util

import (
	"strings"
	"testing"

	"ywwzwb/imagespider/models/config"

	"github.com/PuerkitoBio/goquery"
)

func newTestDoc(t *testing.T, html string) *goquery.Document {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse html: %v", err)
	}
	return doc
}

func TestHTMLParserPrefersPrimarySelector(t *testing.T) {
	doc := newTestDoc(t, `
		<a id="highres" href="https://example.com/jpeg/x.jpg">jpeg</a>
		<a id="png" href="https://example.com/image/x.png">png</a>
	`)
	parser := NewParser(&config.HTMLParserConfig{
		Selector:         "#png",
		FallbackSelector: "#highres",
		Value:            config.ValueConfig{Attribute: config.AttributeTypeHref},
	})

	got, err := parser.Parse(doc)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got) != 1 || got[0] != "https://example.com/image/x.png" {
		t.Fatalf("got %v, want [https://example.com/image/x.png]", got)
	}
}

func TestHTMLParserUsesFallbackSelector(t *testing.T) {
	doc := newTestDoc(t, `
		<a id="highres" href="https://example.com/image/x.jpg">jpeg</a>
	`)
	parser := NewParser(&config.HTMLParserConfig{
		Selector:         "#png",
		FallbackSelector: "#highres",
		Value:            config.ValueConfig{Attribute: config.AttributeTypeHref},
	})

	got, err := parser.Parse(doc)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got) != 1 || got[0] != "https://example.com/image/x.jpg" {
		t.Fatalf("got %v, want [https://example.com/image/x.jpg]", got)
	}
}

package web

import (
	"embed"
	"html/template"
)

// Assets contains public landing-page assets and the management frontend bundle.
//
//go:embed landing admin
var Assets embed.FS

// TemplateFiles contains all HTML templates rendered by the backend.
//
//go:embed templates
var TemplateFiles embed.FS

// LoadTemplates parses every template under templates/ and returns the root set.
// Templates share a consistent look via /assets/landing/gravitylink-landing.css.
func LoadTemplates() (*template.Template, error) {
	return template.ParseFS(TemplateFiles, "templates/*.html")
}

// MustLoadTemplates is a convenience wrapper that panics on parse error.
// Used at startup so configuration errors surface immediately.
func MustLoadTemplates() *template.Template {
	t, err := LoadTemplates()
	if err != nil {
		panic(err)
	}
	return t
}

package router

import (
	"bytes"
	"gravitylink/backend/internal/model"
	"strings"
	"testing"
)

func TestCardTemplateEscapesUntrustedContent(t *testing.T) {
	var out bytes.Buffer
	if e := shareTemplate.Execute(&out, model.ShareCard{Title: `</script><script>alert(1)</script>`, Description: `<img src=x onerror=alert(1)>`, ImageURL: "https://example.com/a.png", TargetURL: "https://example.com"}); e != nil {
		t.Fatal(e)
	}
	if strings.Contains(out.String(), "<script>alert(1)</script>") || strings.Contains(out.String(), "<img src=x onerror") {
		t.Fatal("unsafe HTML rendered")
	}
}

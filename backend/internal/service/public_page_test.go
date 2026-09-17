package service

import (
	"encoding/base64"
	"html/template"
	"strings"
	"testing"

	"gravitylink/backend/internal/model"
)

func TestSanitizeHomeRedirectURL(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"https://five-plus-one.com/", "https://five-plus-one.com/"},
		{"  https://five-plus-one.com/path?x=1  ", "https://five-plus-one.com/path?x=1"},
		{`"https://five-plus-one.com/"`, "https://five-plus-one.com/"},
		{`'https://five-plus-one.com/'`, "https://five-plus-one.com/"},
		{`"https://five-plus-one.com/"`, "https://five-plus-one.com/"},
		{"", ""},
		{"   ", ""},
		{`""`, ""},
		{`javascript:alert(1)`, ""},
		{"/relative/path", ""},
		{"ftp://example.com/", ""},
		{"not a url", ""},
	}
	for _, c := range cases {
		if got := sanitizeHomeRedirectURL(c.in); got != c.want {
			t.Fatalf("sanitizeHomeRedirectURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRenderHomeRedirectUsesSingleJSQuote(t *testing.T) {
	// 回归：html/template 在 script 上下文自动编码字符串，
	// 不得再用 strconv.Quote，否则 location.replace 收到带引号的相对路径。
	tmpl := template.Must(template.New("t").Parse(`<script>location.replace({{.RedirectURL}});</script>`))
	var b strings.Builder
	if err := tmpl.Execute(&b, map[string]string{"RedirectURL": "https://five-plus-one.com/"}); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if !strings.Contains(out, `location.replace("https://five-plus-one.com/")`) {
		t.Fatalf("unexpected JS output: %s", out)
	}
	if strings.Contains(out, `\"https://`) || strings.Contains(out, `location.replace("\"`) {
		t.Fatalf("double-encoded redirect URL: %s", out)
	}
}

func TestValidHomeMode(t *testing.T) {
	for _, mode := range []string{"default", "redirect", "landing"} {
		if !validHomeMode(mode) {
			t.Fatalf("expected %q to be valid", mode)
		}
	}
	for _, mode := range []string{"", "portal", "DEFAULT"} {
		if validHomeMode(mode) {
			t.Fatalf("expected %q to be invalid", mode)
		}
	}
}

func TestHomeModeConstants(t *testing.T) {
	if model.HomeModeDefault != "default" || model.HomeModeRedirect != "redirect" || model.HomeModeLanding != "landing" {
		t.Fatal("home mode constants drifted")
	}
}

func TestLegacyHashScriptContainsDecoder(t *testing.T) {
	for _, needle := range []string{"location.hash", "atob", "location.replace", "^[A-Za-z0-9_-]{2,64}$"} {
		if !strings.Contains(legacyHashScript, needle) {
			t.Fatalf("legacyHashScript missing %s", needle)
		}
	}
}

func TestLegacyHashVectorTDlBa2Y(t *testing.T) {
	// 用户反馈旧版链接：#TDlBa2Y= → L9Akf
	decoded, err := base64.StdEncoding.DecodeString("TDlBa2Y=")
	if err != nil || string(decoded) != "L9Akf" {
		t.Fatal("decode failed", err, string(decoded))
	}
	enc := base64.StdEncoding.EncodeToString([]byte("L9Akf"))
	if enc != "TDlBa2Y=" {
		t.Fatalf("encode mismatch: %s", enc)
	}
}

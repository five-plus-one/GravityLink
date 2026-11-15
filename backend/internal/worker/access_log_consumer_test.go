package worker

import (
	"gravitylink/backend/internal/service"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestAccessLogDescriptionLimits(t *testing.T) {
	r := accessLogFromEvent(service.AccessEvent{ISP: strings.Repeat("运营商", 40), Referer: strings.Repeat("界", 2100), SourceApp: "微信"})
	if r.ISP == nil || utf8.RuneCountInString(*r.ISP) != 64 || !utf8.ValidString(*r.ISP) {
		t.Fatal("ISP length and Unicode")
	}
	if r.Referer == nil || utf8.RuneCountInString(*r.Referer) != 2048 {
		t.Fatal("referer length")
	}
	if r.SourceApp == nil || *r.SourceApp != "微信" {
		t.Fatal("source app")
	}
	if boundedLogString("", 64) != nil {
		t.Fatal("empty value")
	}
}

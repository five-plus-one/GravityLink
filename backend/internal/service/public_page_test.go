package service

import (
	"encoding/base64"
	"strings"
	"testing"
)

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

package service

import (
	"bytes"
	"gravitylink/backend/internal/model"
	"testing"
)

func TestWechatOfficialSignatureVector(t *testing.T) {
	got := WechatSignature("sM4AOVdWfPE4DxkXGEs8VMCPGGVi4C3VM0P37wVUCFvkVAy_90u5h9nbSlYy3-Sl-HhTdfl2fzFy1AOcHKP7qg", "Wm3WZYTPz0wzccnW", 1414587457, "http://mp.weixin.qq.com?params=value#ignored")
	if got != "0f9de62fce790f9a083d5c99e95740ceb90c27ed" {
		t.Fatalf("unexpected signature: %s", got)
	}
}
func TestSecretEncryptionRejectsTamperAndWrongKey(t *testing.T) {
	key := bytes.Repeat([]byte{1}, 32)
	value, e := sealSecret(key, "test-only-secret")
	if e != nil {
		t.Fatal(e)
	}
	plain, e := openSecret(key, value)
	if e != nil || plain != "test-only-secret" {
		t.Fatal("roundtrip failed")
	}
	if _, e = openSecret(bytes.Repeat([]byte{2}, 32), value); e == nil {
		t.Fatal("wrong key accepted")
	}
	if _, e = openSecret(key, value[:len(value)-4]); e == nil {
		t.Fatal("truncated ciphertext accepted")
	}
}
func TestDomainPreservesPortAndRejectsURLComponents(t *testing.T) {
	d, e := domainFromInput(DomainInput{Host: "localhost:18080", Type: "entry", Scheme: "http"})
	if e != nil || d.Host != "localhost:18080" {
		t.Fatal("port lost", e)
	}
	for _, host := range []string{"example.com/path", "user@example.com", "example.com?query", "localhost:70000", "example.com#fragment"} {
		if validDomainHost(host) {
			t.Errorf("accepted %s", host)
		}
	}
	c := &DomainCache{domains: map[string]model.Domain{"localhost:18080": d}}
	if _, ok := c.Get("localhost:18080"); !ok {
		t.Fatal("explicit port missing")
	}
	if _, ok := c.Get("localhost:18081"); ok {
		t.Fatal("wrong port matched")
	}
}

package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStoragePut(t *testing.T) {
	var got []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" || r.URL.Path != "/images/img/photo.png" {
			t.Errorf("unexpected target %s %s", r.Method, r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ") {
			t.Error("missing signature")
		}
		got, _ = io.ReadAll(r.Body)
		w.Header().Set("ETag", "test")
		w.WriteHeader(200)
	}))
	defer server.Close()
	cfg := StorageConfig{Enabled: true, Endpoint: server.URL, Region: "test", Bucket: "images", Prefix: "img", CDN: "https://cdn.example.com", AccessKey: "test", SecretKey: "test-secret", PathStyle: true}
	address, err := storagePut(context.Background(), cfg, "photo.png", []byte("image"))
	if err != nil {
		t.Fatal(err)
	}
	if address != "https://cdn.example.com/img/photo.png" || string(got) != "image" {
		t.Fatal("upload or CDN URL failed")
	}
}
func TestLinkLabels(t *testing.T) {
	c, tags, e := normalizeLinkLabels(" 活动 ", []string{" 测试 ", "测试", ""})
	if e != nil || c != "活动" || len(tags) != 1 {
		t.Fatal(c, tags, e)
	}
	if _, _, e = normalizeLinkLabels(strings.Repeat("字", 81), nil); e == nil {
		t.Fatal("oversized category accepted")
	}
}

func TestOSSConfiguration(t *testing.T) {
	c := normalizeStorage(StorageConfig{Enabled: true, Endpoint: "https://oss-cn-shanghai.aliyuncs.com", Region: "oss-cn-shanghai", Bucket: "photos", CDN: "https://cdn.example.com", AccessKey: "test", SecretKey: "test", PathStyle: true})
	if c.Endpoint != "https://s3.oss-cn-shanghai.aliyuncs.com" || c.Region != "cn-shanghai" || c.PathStyle {
		t.Fatal("OSS configuration not normalized")
	}
	if err := validateStorage(c); err != nil {
		t.Fatal(err)
	}
	c.Region = "photos"
	if err := validateStorage(c); err == nil {
		t.Fatal("bucket name accepted as region")
	}
}

func TestConfigForOverwrite(t *testing.T) {
	ok := StorageConfig{SecretKey: "saved-secret"}
	got, err := configForOverwrite(ok, nil, "")
	if err != nil || got.SecretKey != "saved-secret" {
		t.Fatal("decryptable config should pass through")
	}
	if _, err = configForOverwrite(StorageConfig{}, ErrStorageUndecryptable, "  "); !errors.Is(err, ErrStorageUndecryptable) {
		t.Fatal("undecryptable without secret must fail")
	}
	dbErr := errors.New("db down")
	if _, err = configForOverwrite(StorageConfig{}, dbErr, "new-secret"); !errors.Is(err, dbErr) {
		t.Fatal("non-decrypt errors must surface")
	}
	got, err = configForOverwrite(StorageConfig{}, ErrStorageUndecryptable, "new-secret")
	if err != nil || got.SecretKey != "" {
		t.Fatal("undecryptable with form secret should recover with empty old")
	}
}

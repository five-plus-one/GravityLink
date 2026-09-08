package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminHTMLAndMissingAssets(t *testing.T) {
	handler := NewAdminFrontendHandler(http.NotFoundHandler())
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest("GET", "/domains", nil))
	if page.Code != 200 || page.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("admin entry must not retain stale application references")
	}
	asset := httptest.NewRecorder()
	handler.ServeHTTP(asset, httptest.NewRequest("GET", "/assets/removed-build.js", nil))
	if asset.Code != 404 {
		t.Fatal("missing module must not return SPA HTML")
	}
}

func TestAdminForwardsLandingAssets(t *testing.T) {
	handler := NewAdminFrontendHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/assets/landing/gravitylink-landing.css" {
			t.Fatal("path changed")
		}
		w.Header().Set("Content-Type", "text/css")
		w.WriteHeader(200)
	}))
	result := httptest.NewRecorder()
	handler.ServeHTTP(result, httptest.NewRequest("GET", "/assets/landing/gravitylink-landing.css", nil))
	if result.Code != 200 || result.Header().Get("Content-Type") != "text/css" {
		t.Fatal("landing preview stylesheet unavailable")
	}
}

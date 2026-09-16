package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetHeadRegistersBothMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/")
	handler := func(c *gin.Context) { c.String(http.StatusOK, "ok") }
	getHead(g, "/ping", handler)
	getHead(g, "/:code", handler)

	for _, method := range []string{http.MethodGet, http.MethodHead} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/ping", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s /ping = %d", method, w.Code)
		}
		w = httptest.NewRecorder()
		req = httptest.NewRequest(method, "/abc123", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s /:code = %d", method, w.Code)
		}
	}
}

func TestGetHeadDoesNotBreakNoRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/")
	getHead(g, "/only", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.NoRoute(func(c *gin.Context) { c.Status(http.StatusNotFound) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodHead, "/missing", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("HEAD missing = %d", w.Code)
	}
}

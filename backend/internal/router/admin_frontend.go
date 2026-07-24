package router

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"gravitylink/backend/internal/web"
)

func NewAdminFrontendHandler(api http.Handler) http.Handler {
	adminFS, err := fs.Sub(web.Assets, "admin")
	if err != nil {
		return http.NotFoundHandler()
	}

	fileServer := http.FileServer(http.FS(adminFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			api.ServeHTTP(w, r)
			return
		}

		cleanPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if cleanPath == "." {
			cleanPath = "index.html"
		}

		if info, err := fs.Stat(adminFS, cleanPath); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		r.URL.Path = "/index.html"
		fileServer.ServeHTTP(w, r)
	})
}

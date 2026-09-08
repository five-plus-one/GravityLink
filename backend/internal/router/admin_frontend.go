package router

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"gravitylink/backend/internal/web"
)

func NewAdminFrontendHandler(api http.Handler, setupHandlers ...http.Handler) http.Handler {
	adminFS, err := fs.Sub(web.Assets, "admin")
	if err != nil {
		return http.NotFoundHandler()
	}

	fileServer := http.FileServer(http.FS(adminFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/setup/") && len(setupHandlers) > 0 && setupHandlers[0] != nil {
			setupHandlers[0].ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/uploads/") || strings.HasPrefix(r.URL.Path, "/assets/landing/") {
			api.ServeHTTP(w, r)
			return
		}

		cleanPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if cleanPath == "." {
			serveAdminIndex(w, r, fileServer)
			return
		}

		if info, err := fs.Stat(adminFS, cleanPath); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(cleanPath, "assets/") {
			http.NotFound(w, r)
			return
		}

		serveAdminIndex(w, r, fileServer)
	})
}

func serveAdminIndex(w http.ResponseWriter, _ *http.Request, _ http.Handler) {
	indexHTML, err := fs.ReadFile(web.Assets, "admin/index.html")
	if err != nil {
		http.Error(w, "admin frontend not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(indexHTML)
}

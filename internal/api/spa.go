package api

import (
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// SPAHandler serves static files from an fs.FS, falling back to index.html
// for any path that doesn't match a real file (client-side routing support).
func SPAHandler(fsys fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Check if the file exists in the embedded filesystem.
		if _, err := fs.Stat(fsys, path); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// Fallback: serve index.html for client-side routing.
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

// devProxyHandler returns a reverse-proxy handler that forwards requests
// to the Vite dev server for HMR support during development.
func devProxyHandler(viteURL string) http.Handler {
	target, err := url.Parse(viteURL)
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "invalid dev proxy URL: "+err.Error(), http.StatusBadGateway)
		})
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	return proxy
}

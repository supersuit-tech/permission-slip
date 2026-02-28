package main

import (
	"io/fs"
	"net/http"
)

// spaHandler serves static files from the embedded filesystem.
// If a requested file doesn't exist, it falls back to index.html
// so that client-side routing works.
func spaHandler(staticFS fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(staticFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to find the file in the static FS
		path := r.URL.Path
		if path != "/" {
			if _, err := fs.Stat(staticFS, path[1:]); err != nil {
				// File not found — serve index.html for SPA routing
				r.URL.Path = "/"
			}
		}
		fileServer.ServeHTTP(w, r)
	})
}

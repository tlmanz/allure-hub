package transport

import (
	"net/http"
	"path"
	"path/filepath"
	"strings"
)

func reportsHandler(dataDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/reports/")
		parts := strings.SplitN(p, "/", 3)
		if len(parts) < 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
			http.NotFound(w, r)
			return
		}
		envID := parts[0]
		projectID := parts[1]
		rest := parts[2]

		target := filepath.Join(dataDir, envID, projectID, "reports", rest)
		absTarget, err := filepath.Abs(target)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		absBase, err := filepath.Abs(filepath.Join(dataDir, envID, projectID, "reports"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if !strings.HasPrefix(absTarget, absBase+string(filepath.Separator)) && absTarget != absBase {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self' 'unsafe-inline' 'unsafe-eval' blob: data: https:; "+
				"img-src 'self' data: blob: https:; "+
				"font-src 'self' data: https:; "+
				"connect-src 'self' data: blob: https:; "+
				"worker-src blob: 'self'; "+
				"frame-ancestors 'self';",
		)
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		http.ServeFile(w, r, absTarget)
	})
}

func spaHandler(webDir string) http.Handler {
	fsys := http.Dir(webDir)
	fileServer := http.FileServer(fsys)
	indexPath := filepath.Join(webDir, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cleanPath := path.Clean("/" + r.URL.Path)
		f, err := fsys.Open(cleanPath)
		if err != nil {
			http.ServeFile(w, r, indexPath)
			return
		}
		f.Close()
		fileServer.ServeHTTP(w, r)
	})
}

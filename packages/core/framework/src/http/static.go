package http

import (
	"mime"
	nethttp "net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Static mirrors express.static(dir) mounted at mount: GET/HEAD requests
// for existing files are served, anything else goes to the next layer.
func Static(mount, dir string) MiddlewareFunction {
	return func(req *MedusaRequest, res *MedusaResponse, next NextFunction) error {
		if req.Method != nethttp.MethodGet && req.Method != nethttp.MethodHead {
			next(nil)
			return nil
		}
		rel := strings.TrimPrefix(req.URL.Path, mount)
		if rel == "" || strings.HasSuffix(rel, "/") {
			next(nil)
			return nil
		}
		clean := path.Clean("/" + rel)
		p := filepath.Join(dir, filepath.FromSlash(clean))
		f, err := os.Open(p)
		if err != nil {
			next(nil)
			return nil
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil || info.IsDir() {
			next(nil)
			return nil
		}
		w := res.Handoff()
		w.Header().Set("Content-Type", sendContentType(info.Name()))
		w.Header().Set("Cache-Control", "public, max-age=0")
		w.Header().Set("Accept-Ranges", "bytes")
		nethttp.ServeContent(w, req.Request, info.Name(), info.ModTime(), f)
		res.end()
		return nil
	}
}

// sendContentType mirrors send's Content-Type: mime.lookup(path) with the
// UTF-8 charset of text and JSON/JavaScript types.
func sendContentType(name string) string {
	t := mime.TypeByExtension(filepath.Ext(name))
	if t == "" {
		return "application/octet-stream"
	}
	if i := strings.Index(t, ";"); i >= 0 {
		t = strings.TrimSpace(t[:i])
	}
	if strings.HasPrefix(t, "text/") || t == "application/javascript" || t == "application/json" {
		return t + "; charset=UTF-8"
	}
	return t
}

// Package commands mirrors @medusajs/admin-bundler/src/commands for the Go
// server: serving a built admin dashboard (serve.ts) and configuring the
// prebuilt "runtime" bundles produced by build-admin.mjs.
package commands

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io/fs"
	"math"
	"mime"
	nethttp "net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
)

const (
	// BasePlaceholder is the Vite base of runtime bundles.
	BasePlaceholder = "/__MEDUSA_ADMIN_BASE__"
	// RuntimeMarker marks the place in index.html of a runtime bundle where
	// the configuration script is inserted.
	RuntimeMarker = "<!--__MEDUSA_ADMIN_RUNTIME_CONFIG__-->"
)

// BundlerOptions mirrors the BundlerOptions values the admin uses at run
// time (the Vite `define`s of getViteConfig).
type BundlerOptions struct {
	Path              string
	BackendURL        string
	StorefrontURL     string
	MaxUploadFileSize float64
}

// Bundle is a built admin dashboard held in memory.
type Bundle struct {
	html  []byte
	files map[string]*bundleFile
}

type bundleFile struct {
	data   []byte
	ctype  string
	etag   string
	gzOnce sync.Once
	gz     []byte
}

// textExtensions are the files in which the base placeholder is replaced.
var textExtensions = map[string]bool{
	".html": true, ".js": true, ".mjs": true, ".css": true, ".json": true,
	".svg": true, ".map": true, ".txt": true,
}

// normalizedBase mirrors Vite's base normalization ("/app" -> "/app/").
func normalizedBase(p string) string {
	if strings.HasSuffix(p, "/") {
		return p
	}
	return p + "/"
}

// runtimeScript is the inline script that defines, for a runtime bundle, the
// values getViteConfig compiles in with `define`.
func runtimeScript(o BundlerOptions) string {
	js := func(v any) string {
		raw, _ := json.Marshal(v)
		return string(raw)
	}
	optionalEnv := func(key string) string {
		if v, ok := os.LookupEnv(key); ok {
			return js(v)
		}
		return "undefined"
	}
	maxUpload := "Infinity"
	if !math.IsInf(o.MaxUploadFileSize, 1) {
		maxUpload = strconv.FormatFloat(o.MaxUploadFileSize, 'f', -1, 64)
	}
	// process.env: BACKEND_URL plus the PLUGIN_* variables without prefix.
	pluginEnv := map[string]string{"BACKEND_URL": o.BackendURL}
	for _, kv := range os.Environ() {
		k, v, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(k, "PLUGIN_") {
			pluginEnv[strings.TrimPrefix(k, "PLUGIN_")] = v
		}
	}
	return "<script>var " +
		"__MEDUSA_ADMIN_PATH__=" + js(o.Path) +
		",__MEDUSA_ADMIN_BACKEND_URL__=" + js(o.BackendURL) +
		",__MEDUSA_ADMIN_STOREFRONT_URL__=" + js(o.StorefrontURL) +
		",__MEDUSA_ADMIN_AUTH_TYPE__=" + optionalEnv("ADMIN_AUTH_TYPE") +
		",__MEDUSA_ADMIN_JWT_TOKEN_STORAGE_KEY__=" + optionalEnv("ADMIN_JWT_TOKEN_STORAGE_KEY") +
		",__MEDUSA_ADMIN_MAX_UPLOAD_FILE_SIZE__=" + maxUpload +
		",__MEDUSA_ADMIN_PROCESS_ENV__=" + js(pluginEnv) +
		";</script>"
}

// configure applies the options to one file of a runtime bundle.
func configure(name string, data []byte, o BundlerOptions) []byte {
	if !textExtensions[strings.ToLower(path.Ext(name))] {
		return data
	}
	out := bytes.ReplaceAll(data, []byte(BasePlaceholder+"/"), []byte(normalizedBase(o.Path)))
	out = bytes.ReplaceAll(out, []byte(BasePlaceholder), []byte(o.Path))
	if name == "index.html" {
		out = bytes.Replace(out, []byte(RuntimeMarker), []byte(runtimeScript(o)), 1)
	}
	return out
}

// IsRuntimeBundle reports whether fsys holds a runtime bundle (built with
// mode "runtime"), which is configured when served.
func IsRuntimeBundle(fsys fs.FS) bool {
	html, err := fs.ReadFile(fsys, "index.html")
	return err == nil && bytes.Contains(html, []byte(RuntimeMarker))
}

// LoadBundle reads a built admin. Runtime bundles are configured with o;
// bundles built with compiled-in options are kept as they are.
func LoadBundle(fsys fs.FS, o BundlerOptions) (*Bundle, error) {
	html, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		return nil, errors.New("Could not find index.html in the admin build directory. Make sure to run 'medusa build' before starting the server.")
	}
	runtime := bytes.Contains(html, []byte(RuntimeMarker))
	b := &Bundle{files: map[string]*bundleFile{}}
	err = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		if runtime {
			data = configure(p, data, o)
		}
		b.files[p] = &bundleFile{data: data, ctype: contentType(p), etag: etag(data)}
		return nil
	})
	if err != nil {
		return nil, err
	}
	b.html = b.files["index.html"].data
	return b, nil
}

// WriteTo writes the bundle (configured, with the options compiled in) to
// dir, which is emptied first.
func (b *Bundle) WriteTo(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	for name, f := range b.files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(p, f.data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// CopyFS copies fsys to dir as it is (dir is emptied first).
func CopyFS(fsys fs.FS, dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return os.CopyFS(dir, fsys)
}

func etag(data []byte) string {
	return fmt.Sprintf(`W/"%x-%08x"`, len(data), crc32.ChecksumIEEE(data))
}

func contentType(name string) string {
	t := mime.TypeByExtension(path.Ext(name))
	if t == "" {
		return "application/octet-stream"
	}
	if strings.HasPrefix(t, "text/") || strings.Contains(t, "javascript") || strings.Contains(t, "json") {
		if !strings.Contains(t, "charset") {
			t += "; charset=utf-8"
		}
	}
	return t
}

// compressible mirrors the compression middleware filter (compressible
// content types) for the files of a dashboard build.
func compressible(ctype string) bool {
	return strings.HasPrefix(ctype, "text/") || strings.Contains(ctype, "javascript") ||
		strings.Contains(ctype, "json") || strings.Contains(ctype, "svg") ||
		strings.Contains(ctype, "font/ttf") || strings.Contains(ctype, "font/otf")
}

func (f *bundleFile) gzipped() []byte {
	f.gzOnce.Do(func() {
		var buf bytes.Buffer
		zw, _ := gzip.NewWriterLevel(&buf, gzip.DefaultCompression)
		_, _ = zw.Write(f.data)
		_ = zw.Close()
		f.gz = buf.Bytes()
	})
	return f.gz
}

func acceptsGzip(r *nethttp.Request) bool {
	for _, part := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		enc, q, _ := strings.Cut(strings.TrimSpace(part), ";")
		if strings.EqualFold(strings.TrimSpace(enc), "gzip") && strings.TrimSpace(q) != "q=0" {
			return true
		}
	}
	return false
}

func addVary(h nethttp.Header, value string) {
	if v := h.Get("Vary"); v != "" {
		h.Set("Vary", v+", "+value)
		return
	}
	h.Set("Vary", value)
}

// Middleware mirrors the router of serve(): GET / and unknown paths send
// index.html (Cache-Control: no-cache), the build files are served with
// immutable caching, compressed when the client accepts gzip. mount is the
// admin path the middleware is registered on.
func (b *Bundle) Middleware(mount string) medusahttp.MiddlewareFunction {
	mount = strings.TrimSuffix(mount, "/")
	return func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
		if req.Method != nethttp.MethodGet && req.Method != nethttp.MethodHead {
			next(nil)
			return nil
		}
		p := req.URL.Path
		rel := ""
		if len(p) >= len(mount) {
			rel = p[len(mount):]
		}
		if rel != "" && rel != "/" {
			name := strings.TrimPrefix(path.Clean("/"+rel), "/")
			if f, ok := b.files[name]; ok && !strings.HasPrefix(path.Base(name), ".") {
				b.serveFile(req, res, name, f)
				return nil
			}
		}
		res.SetHeader("Cache-Control", "no-cache")
		res.SetHeader("Vary", "Origin, Cache-Control")
		res.Send(string(b.html))
		return nil
	}
}

func (b *Bundle) serveFile(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, name string, f *bundleFile) {
	w := res.Handoff()
	h := w.Header()
	h.Set("Cache-Control", "max-age=31536000, immutable")
	h.Set("Vary", "Origin, Cache-Control")
	h.Set("Content-Type", f.ctype)
	h.Set("ETag", f.etag)
	if compressible(f.ctype) && len(f.data) >= 1024 {
		addVary(h, "Accept-Encoding")
		if acceptsGzip(req.Request) {
			if inm := req.Header.Get("If-None-Match"); inm != "" && strings.Contains(inm, f.etag) {
				w.WriteHeader(nethttp.StatusNotModified)
				return
			}
			gz := f.gzipped()
			h.Set("Content-Encoding", "gzip")
			h.Set("Content-Length", strconv.Itoa(len(gz)))
			w.WriteHeader(nethttp.StatusOK)
			if req.Method != nethttp.MethodHead {
				_, _ = w.Write(gz)
			}
			return
		}
	}
	h.Set("Accept-Ranges", "bytes")
	nethttp.ServeContent(w, req.Request, path.Base(name), zeroTime, bytes.NewReader(f.data))
}

var zeroTime time.Time

// ServeOptions mirrors ServeOptions of serve.ts.
type ServeOptions struct {
	// OutDir is the admin build directory (<root>/public/admin).
	OutDir string
	// Fallback is served when OutDir holds no build (the embedded prebuilt
	// dashboard). Without it a missing build is an error, like upstream.
	Fallback fs.FS
	BundlerOptions
}

// fallbackCache keeps the configured fallback bundle per configuration.
var fallbackCache sync.Map

// Serve mirrors serve(options): it returns the middleware serving the admin
// build of options.OutDir, or the fallback bundle.
func Serve(o ServeOptions) (medusahttp.MiddlewareFunction, error) {
	if o.OutDir != "" {
		if _, err := os.Stat(filepath.Join(o.OutDir, "index.html")); err == nil {
			b, err := LoadBundle(os.DirFS(o.OutDir), o.BundlerOptions)
			if err != nil {
				return nil, err
			}
			return b.Middleware(o.Path), nil
		}
	}
	if o.Fallback == nil {
		return nil, errors.New("Could not find index.html in the admin build directory. Make sure to run 'medusa build' before starting the server.")
	}
	key := runtimeScript(o.BundlerOptions)
	if cached, ok := fallbackCache.Load(key); ok {
		return cached.(*Bundle).Middleware(o.Path), nil
	}
	b, err := LoadBundle(o.Fallback, o.BundlerOptions)
	if err != nil {
		return nil, err
	}
	fallbackCache.Store(key, b)
	return b.Middleware(o.Path), nil
}

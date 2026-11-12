package dashboardbundle

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"embed"
	"io"
	"io/fs"
	"sync"
	"testing/fstest"
)

// Variants of the dashboard with the admin extension of an optional plugin
// (app-<plugin dir>.tar.gz, built by scripts/build-admin-bundle.sh with the
// default plugins and the plugin).
//
//go:embed app-*.tar.gz
var variants embed.FS

// variantArchives maps the package name of an optional plugin to its
// dashboard variant.
var variantArchives = map[string]string{
	"@medusajs/loyalty-plugin": "app-loyalty.tar.gz",
}

var (
	variantsMu  sync.Mutex
	variantsFSs = map[string]fs.FS{}
)

// AppFor returns the dashboard for the configured plugins: the variant of
// an optional plugin with an admin extension, else the default one.
func AppFor(plugins []string) (fs.FS, error) {
	for _, name := range plugins {
		archive, ok := variantArchives[name]
		if !ok {
			continue
		}
		variantsMu.Lock()
		defer variantsMu.Unlock()
		if v, ok := variantsFSs[archive]; ok {
			return v, nil
		}
		v, err := untar(archive)
		if err != nil {
			return nil, err
		}
		variantsFSs[archive] = v
		return v, nil
	}
	return App(), nil
}

func untar(name string) (fs.FS, error) {
	raw, err := variants.ReadFile(name)
	if err != nil {
		return nil, err
	}
	gz, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	out := fstest.MapFS{}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, err
		}
		out[h.Name] = &fstest.MapFile{Data: data, Mode: 0o444}
	}
}

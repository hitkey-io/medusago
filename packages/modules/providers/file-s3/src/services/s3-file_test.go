package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	filetypes "github.com/hitkey-io/medusago/packages/core/types/src/file"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
)

// Vectors captured from upstream by scripts/extract-provider-vectors.js
// (fixed clock and ULID, the SDK client's send replaced by a recorder).
type s3Vectors struct {
	FixedDate string                    `json:"fixed_date"`
	Configs   map[string]map[string]any `json:"configs"`
	Presigned []struct {
		Config    string  `json:"config"`
		Method    string  `json:"method"`
		FileKey   string  `json:"fileKey"`
		URL       string  `json:"url"`
		Filename  string  `json:"filename"`
		Access    *string `json:"access"`
		ExpiresIn *int    `json:"expiresIn"`
		Result    struct {
			URL string `json:"url"`
			Key string `json:"key"`
		} `json:"result"`
	} `json:"presigned"`
	Uploads []struct {
		Config   string  `json:"config"`
		Filename string  `json:"filename"`
		Content  string  `json:"content"`
		MimeType *string `json:"mimeType"`
		Access   *string `json:"access"`
		Result   struct {
			URL string `json:"url"`
			Key string `json:"key"`
		} `json:"result"`
		Sent []struct {
			Command string `json:"command"`
			Input   struct {
				ACL          *string           `json:"ACL"`
				Bucket       string            `json:"Bucket"`
				Body         string            `json:"Body"`
				Key          string            `json:"Key"`
				ContentType  *string           `json:"ContentType"`
				CacheControl string            `json:"CacheControl"`
				Metadata     map[string]string `json:"Metadata"`
			} `json:"input"`
		} `json:"sent"`
	} `json:"uploads"`
	Errors []struct {
		Config   string            `json:"config"`
		Method   string            `json:"method"`
		Filename *string           `json:"filename"`
		Access   *string           `json:"access"`
		Args     []json.RawMessage `json:"args"`
		Type     string            `json:"type"`
		Message  string            `json:"message"`
	} `json:"errors"`
}

const fixedULID = "01KDYZZZZZZZZZZZZZZZZZZZZZ"

func loadS3Vectors(t *testing.T) s3Vectors {
	t.Helper()
	raw, err := os.ReadFile("testdata/upstream-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var v s3Vectors
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func str(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// recordedRequest is a request the client sent (on any host).
type recordedRequest struct {
	method string
	url    *url.URL
	header http.Header
	body   []byte
}

// redirectTransport sends every request to the test server, keeping the
// original URL.
type redirectTransport struct {
	mu       sync.Mutex
	target   *url.URL
	requests []recordedRequest
}

func (rt *redirectTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	body, _ := io.ReadAll(r.Body)
	rt.mu.Lock()
	rt.requests = append(rt.requests, recordedRequest{method: r.Method, url: r.URL, header: r.Header.Clone(), body: body})
	rt.mu.Unlock()
	return &http.Response{StatusCode: 200, Body: http.NoBody, Header: http.Header{}, Request: r}, nil
}

func newVectorService(t *testing.T, v s3Vectors, name string) (*S3FileService, *redirectTransport) {
	t.Helper()
	s, err := NewS3FileService(v.Configs[name], nil)
	if err != nil {
		t.Fatal(err)
	}
	fixed, _ := time.Parse(time.RFC3339Nano, v.FixedDate)
	s.client.now = func() time.Time { return fixed }
	rt := &redirectTransport{}
	s.client.httpClient = &http.Client{Transport: rt}
	return s, rt
}

func TestPresignedURLsMatchUpstream(t *testing.T) {
	v := loadS3Vectors(t)
	ctx := context.Background()
	for _, p := range v.Presigned {
		s, _ := newVectorService(t, v, p.Config)
		if p.Method == "download" {
			got, err := s.GetPresignedDownloadURL(ctx, p.FileKey)
			if err != nil || got != p.URL {
				t.Errorf("%s download %q:\n got %s (%v)\nwant %s", p.Config, p.FileKey, got, err, p.URL)
			}
			continue
		}
		got, err := s.GetPresignedUploadURL(ctx, filetypes.ProviderGetPresignedUploadUrlDTO{Filename: p.Filename, MimeType: "image/png", Access: str(p.Access), ExpiresIn: p.ExpiresIn})
		if err != nil || got.URL != p.Result.URL || got.Key != p.Result.Key {
			t.Errorf("%s upload %q %v:\n got %s %s (%v)\nwant %s %s", p.Config, p.Filename, str(p.Access), got.Key, got.URL, err, p.Result.Key, p.Result.URL)
		}
	}
}

func TestUploadsMatchUpstream(t *testing.T) {
	v := loadS3Vectors(t)
	ctx := context.Background()
	prev := newULID
	newULID = func() string { return fixedULID }
	defer func() { newULID = prev }()
	for _, u := range v.Uploads {
		s, rt := newVectorService(t, v, u.Config)
		got, err := s.Upload(ctx, &filetypes.ProviderUploadFileDTO{Filename: u.Filename, Content: u.Content, MimeType: str(u.MimeType), Access: str(u.Access)})
		if err != nil {
			t.Fatalf("%s %q: %v", u.Config, u.Filename, err)
		}
		if got.Key != u.Result.Key || got.URL != u.Result.URL {
			t.Errorf("%s %q: got %s %s, want %s %s", u.Config, u.Filename, got.Key, got.URL, u.Result.Key, u.Result.URL)
		}
		if len(rt.requests) != 1 || len(u.Sent) != 1 {
			t.Fatalf("%s %q: %d requests", u.Config, u.Filename, len(rt.requests))
		}
		req, in := rt.requests[0], u.Sent[0].Input
		wantURL := s.client.objectURL(in.Bucket, in.Key)
		if req.method != http.MethodPut || req.url.EscapedPath() != wantURL.EscapedPath() || req.url.Host != wantURL.Host || req.url.RawQuery != "x-id=PutObject" {
			t.Errorf("%s %q: request %s %s", u.Config, u.Filename, req.method, req.url)
		}
		checks := map[string]string{
			"X-Amz-Acl":                    str(in.ACL),
			"Content-Type":                 str(in.ContentType),
			"Cache-Control":                in.CacheControl,
			"X-Amz-Meta-Original-Filename": in.Metadata["original-filename"],
		}
		for h, want := range checks {
			if got := req.header.Get(h); got != want {
				t.Errorf("%s %q: %s = %q, want %q", u.Config, u.Filename, h, got, want)
			}
		}
		if base64.StdEncoding.EncodeToString(req.body) != in.Body {
			t.Errorf("%s %q: body %q, want %s", u.Config, u.Filename, req.body, in.Body)
		}
	}
}

func TestErrorsMatchUpstream(t *testing.T) {
	v := loadS3Vectors(t)
	ctx := context.Background()
	for _, e := range v.Errors {
		var err error
		switch {
		case e.Method == "constructor":
			_, err = NewS3FileService(map[string]any{"bucket": "b", "region": "r", "file_url": "u"}, nil)
		case e.Filename != nil:
			s, _ := newVectorService(t, v, e.Config)
			if e.Method == "upload" {
				_, err = s.Upload(ctx, &filetypes.ProviderUploadFileDTO{Filename: *e.Filename, Content: "x", Access: "public"})
			} else {
				_, err = s.GetPresignedUploadURL(ctx, filetypes.ProviderGetPresignedUploadUrlDTO{Filename: *e.Filename, Access: str(e.Access)})
			}
		default:
			s, _ := newVectorService(t, v, e.Config)
			var arg struct {
				Filename string `json:"filename"`
				FileKey  string `json:"fileKey"`
			}
			nilArg := string(e.Args[0]) == "null"
			_ = json.Unmarshal(e.Args[0], &arg)
			switch e.Method {
			case "upload":
				in := &filetypes.ProviderUploadFileDTO{Filename: arg.Filename}
				if nilArg {
					in = nil
				}
				_, err = s.Upload(ctx, in)
			case "getPresignedUploadUrl":
				_, err = s.GetPresignedUploadURL(ctx, filetypes.ProviderGetPresignedUploadUrlDTO{Filename: arg.Filename})
			case "getAsBuffer":
				_, err = s.GetAsBuffer(ctx, arg.FileKey)
			case "getDownloadStream":
				_, err = s.GetDownloadStream(ctx, arg.FileKey)
			case "getUploadStream":
				_, err = s.GetUploadStream(ctx, filetypes.ProviderUploadStreamDTO{Filename: arg.Filename})
			}
		}
		var me *common.MedusaError
		if !errors.As(err, &me) || me.Type != e.Type || me.Message != e.Message {
			t.Errorf("%s %s %v: got %v, want %s %q", e.Config, e.Method, str(e.Filename), err, e.Type, e.Message)
		}
	}
}

func TestNodePath(t *testing.T) {
	for in, want := range map[string]string{
		"x/../y.png": "y.png", "../../etc/passwd": "etc/passwd", "/abs//dir/./f.tar.gz": "abs/dir/f.tar.gz",
		"\\win\\path\\f.TXT": "win/path/f.TXT", "a/b/": "a/b/", "..": "", "./": "", "a/../..": "",
	} {
		if got := sanitizeFilePath(in); got != want {
			t.Errorf("sanitizeFilePath(%q) = %q, want %q", in, got, want)
		}
	}
	for in, want := range map[string]parsedPath{
		"a/b.c.d":   {Dir: "a", Base: "b.c.d", Name: "b.c", Ext: ".d"},
		".hidden":   {Base: ".hidden", Name: ".hidden"},
		"dot.":      {Base: "dot.", Name: "dot", Ext: "."},
		"trailing/": {Base: "trailing", Name: "trailing"},
		"..":        {Base: "..", Name: ".."},
	} {
		if got := posixParse(in); got != want {
			t.Errorf("posixParse(%q) = %+v, want %+v", in, got, want)
		}
	}
}

// TestMinio runs the provider against an S3 compatible server
// (MEDUSA_TEST_S3_ENDPOINT, e.g. a MinIO container with the bucket
// MEDUSA_TEST_S3_BUCKET and credentials MEDUSA_TEST_S3_KEY / _SECRET).
func TestMinio(t *testing.T) {
	endpoint := os.Getenv("MEDUSA_TEST_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("MEDUSA_TEST_S3_ENDPOINT is not set")
	}
	s, err := NewS3FileService(map[string]any{
		"file_url": endpoint + "/" + os.Getenv("MEDUSA_TEST_S3_BUCKET"), "access_key_id": os.Getenv("MEDUSA_TEST_S3_KEY"),
		"secret_access_key": os.Getenv("MEDUSA_TEST_S3_SECRET"), "region": "us-east-1", "bucket": os.Getenv("MEDUSA_TEST_S3_BUCKET"),
		"endpoint": endpoint, "prefix": "go-test/", "additional_client_config": map[string]any{"forcePathStyle": true},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	up, err := s.Upload(ctx, &filetypes.ProviderUploadFileDTO{Filename: "dir/hello wörld+.txt", Content: "hello ünïcode", MimeType: "text/plain", Access: "public"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.GetAsBuffer(ctx, up.Key)
	if err != nil || string(got) != "hello ünïcode" {
		t.Fatalf("GetAsBuffer = %q, %v", got, err)
	}
	// a multipart upload (more than one 5 MiB part)
	stream, err := s.GetUploadStream(ctx, filetypes.ProviderUploadStreamDTO{Filename: "big.csv", MimeType: "text/csv"})
	if err != nil {
		t.Fatal(err)
	}
	chunk := make([]byte, 1024*1024)
	for i := range chunk {
		chunk[i] = 'a' + byte(i%26)
	}
	for i := 0; i < 11; i++ {
		if _, err := stream.File.Write(chunk); err != nil {
			t.Fatal(err)
		}
	}
	if err := stream.File.Close(); err != nil {
		t.Fatal(err)
	}
	big, err := s.GetAsBuffer(ctx, stream.FileKey)
	if err != nil || len(big) != 11*len(chunk) {
		t.Fatalf("multipart: %d bytes, %v", len(big), err)
	}
	// presigned upload then download (the default URLs carry the CRC32 of an
	// empty body, like upstream: only WHEN_REQUIRED URLs accept content)
	s.client.checksumWhenRequired = true
	pre, err := s.GetPresignedUploadURL(ctx, filetypes.ProviderGetPresignedUploadUrlDTO{Filename: "pre.txt", Access: "public"})
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodPut, pre.URL, nil)
	req.Body = io.NopCloser(strings.NewReader("presigned"))
	req.ContentLength = int64(len("presigned"))
	res, err := http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != 200 {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("presigned put: %v %s", err, body)
	}
	dl, _ := s.GetPresignedDownloadURL(ctx, pre.Key)
	res, err = http.Get(dl)
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("presigned get: %v %v", err, res.Status)
	}
	if b, _ := io.ReadAll(res.Body); string(b) != "presigned" {
		t.Fatalf("presigned content %q", b)
	}
	_ = s.Delete(ctx, []string{up.Key, stream.FileKey, pre.Key})
	if _, err := s.GetAsBuffer(ctx, up.Key); err == nil {
		t.Fatal("deleted object still readable")
	} else if s3err := new(S3Error); !errors.As(err, &s3err) || s3err.Code != "NoSuchKey" {
		t.Fatalf("missing object error: %v", err)
	}
}

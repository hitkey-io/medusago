package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The part of @aws-sdk/client-s3 (and @aws-sdk/s3-request-presigner,
// @aws-sdk/lib-storage) the provider uses: Signature Version 4 signed
// requests and presigned URLs, with the CRC32 flexible checksums and the
// x-id query parameter the JavaScript SDK sends.

const (
	emptySHA256      = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	unsignedPayload  = "UNSIGNED-PAYLOAD"
	sigV4Algorithm   = "AWS4-HMAC-SHA256"
	s3XMLNamespace   = "http://s3.amazonaws.com/doc/2006-03-01/"
	defaultMaxTries  = 3
	minPartSizeBytes = 1024 * 1024 * 5
)

// s3Client is a minimal S3 client.
type s3Client struct {
	region         string
	endpoint       *url.URL // nil: the AWS endpoint of the region
	forcePathStyle bool
	credentials    credentialsProvider
	maxAttempts    int
	httpClient     *http.Client
	now            func() time.Time
	// checksumWhenRequired mirrors requestChecksumCalculation
	// "WHEN_REQUIRED" (default "WHEN_SUPPORTED").
	checksumWhenRequired bool
	// validationWhenRequired mirrors responseChecksumValidation
	// "WHEN_REQUIRED" (default "WHEN_SUPPORTED").
	validationWhenRequired bool
}

const (
	checksumNone = iota
	checksumSupported
	checksumRequired
)

// checksumModeHeaders are the GetObject headers enabling response checksums.
func (c *s3Client) checksumModeHeaders() map[string]string {
	if c.validationWhenRequired {
		return map[string]string{}
	}
	return map[string]string{"x-amz-checksum-mode": "ENABLED"}
}

// S3Error is an error response of S3 (the SDK's S3ServiceException: name is
// the error code, message the error message).
type S3Error struct {
	Code       string
	Message    string
	StatusCode int
}

func (e *S3Error) Error() string { return e.Message }

// dnsCompatibleBucket mirrors isDnsCompatibleBucketName.
var dnsCompatibleBucket = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`)
var ipAddressRe = regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+$`)

func isDNSCompatibleBucketName(bucket string) bool {
	return dnsCompatibleBucket.MatchString(bucket) && !ipAddressRe.MatchString(bucket) && !strings.Contains(bucket, "..") &&
		!strings.Contains(bucket, ".-") && !strings.Contains(bucket, "-.")
}

// objectURL builds the URL of a bucket (key "") or object, path style or
// virtual hosted style like the S3 endpoint ruleset.
func (c *s3Client) objectURL(bucket, key string) *url.URL {
	var base *url.URL
	if c.endpoint != nil {
		u := *c.endpoint
		base = &u
	} else {
		base = &url.URL{Scheme: "https", Host: fmt.Sprintf("s3.%s.amazonaws.com", c.region)}
	}
	encodedKey := encodeS3Key(key)
	pathStyle := c.forcePathStyle || !isDNSCompatibleBucketName(bucket) ||
		(base.Scheme == "https" && strings.Contains(bucket, "."))
	prefix := strings.TrimSuffix(base.EscapedPath(), "/")
	u := &url.URL{Scheme: base.Scheme, Host: base.Host}
	var rawPath string
	if pathStyle {
		rawPath = prefix + "/" + bucket + "/" + encodedKey
	} else {
		u.Host = bucket + "." + base.Host
		rawPath = prefix + "/" + encodedKey
	}
	u.RawPath = rawPath
	u.Path, _ = url.PathUnescape(rawPath)
	return u
}

// encodeS3Key encodes each segment of an object key (extendedEncodeURIComponent).
func encodeS3Key(key string) string {
	segments := strings.Split(key, "/")
	for i, s := range segments {
		segments[i] = uriEncode(s)
	}
	return strings.Join(segments, "/")
}

// uriEncode percent-encodes everything but the RFC 3986 unreserved
// characters.
func uriEncode(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == '~' {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

type queryParam struct{ key, value string }

func canonicalQuery(params []queryParam) string {
	enc := make([]queryParam, len(params))
	for i, p := range params {
		enc[i] = queryParam{uriEncode(p.key), uriEncode(p.value)}
	}
	sort.SliceStable(enc, func(i, j int) bool {
		if enc[i].key != enc[j].key {
			return enc[i].key < enc[j].key
		}
		return enc[i].value < enc[j].value
	})
	parts := make([]string, len(enc))
	for i, p := range enc {
		parts[i] = p.key + "=" + p.value
	}
	return strings.Join(parts, "&")
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// crc32Base64 is the x-amz-checksum-crc32 value of a body.
func crc32Base64(b []byte) string {
	var out [4]byte
	binary.BigEndian.PutUint32(out[:], crc32.ChecksumIEEE(b))
	return base64.StdEncoding.EncodeToString(out[:])
}

func (c *s3Client) signature(creds awsCredentials, t time.Time, canonicalRequest string) string {
	date := t.Format("20060102")
	scope := fmt.Sprintf("%s/%s/s3/aws4_request", date, c.region)
	stringToSign := strings.Join([]string{sigV4Algorithm, t.Format("20060102T150405Z"), scope, sha256Hex([]byte(canonicalRequest))}, "\n")
	key := hmacSHA256([]byte("AWS4"+creds.SecretAccessKey), date)
	key = hmacSHA256(key, c.region)
	key = hmacSHA256(key, "s3")
	key = hmacSHA256(key, "aws4_request")
	return hex.EncodeToString(hmacSHA256(key, stringToSign))
}

// presign mirrors getSignedUrl: the x-amz-* headers of the command are
// hoisted into the query, only host is signed and the payload is unsigned.
func (c *s3Client) presign(ctx context.Context, method, bucket, key string, params []queryParam, expiresIn int) (string, error) {
	creds, err := c.credentials.retrieve(ctx)
	if err != nil {
		return "", err
	}
	t := c.now().UTC()
	u := c.objectURL(bucket, key)
	all := append([]queryParam{
		{"X-Amz-Algorithm", sigV4Algorithm},
		{"X-Amz-Content-Sha256", unsignedPayload},
		{"X-Amz-Credential", fmt.Sprintf("%s/%s/%s/s3/aws4_request", creds.AccessKeyID, t.Format("20060102"), c.region)},
		{"X-Amz-Date", t.Format("20060102T150405Z")},
		{"X-Amz-Expires", strconv.Itoa(expiresIn)},
		{"X-Amz-SignedHeaders", "host"},
	}, params...)
	if creds.SessionToken != "" {
		all = append(all, queryParam{"X-Amz-Security-Token", creds.SessionToken})
	}
	cq := canonicalQuery(all)
	canonical := strings.Join([]string{method, u.EscapedPath(), cq, "host:" + u.Host + "\n", "host", unsignedPayload}, "\n")
	sig := c.signature(creds, t, canonical)
	u.RawQuery = canonicalQuery(append(all, queryParam{"X-Amz-Signature", sig}))
	return u.String(), nil
}

// s3Request is one S3 API call.
type s3Request struct {
	method  string
	bucket  string
	key     string
	query   []queryParam
	headers map[string]string
	body    []byte
	// checksum adds the CRC32 flexible checksum of the body: when
	// supported by the operation (checksumSupported) or always
	// (checksumRequired).
	checksum int
}

// do sends a signed request (retrying 5xx responses and network errors
// like the standard retry strategy) and returns the successful response.
func (c *s3Client) do(ctx context.Context, r s3Request) (*http.Response, error) {
	var lastErr error
	for attempt := 1; attempt <= c.maxAttempts; attempt++ {
		res, err := c.send(ctx, r)
		if err == nil && res.StatusCode < 300 {
			return res, nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = parseS3Error(res)
		}
		var s3err *S3Error
		if errors.As(lastErr, &s3err) && s3err.StatusCode < 500 {
			return nil, lastErr
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if attempt < c.maxAttempts {
			time.Sleep(time.Duration(100*attempt) * time.Millisecond)
		}
	}
	return nil, lastErr
}

func (c *s3Client) send(ctx context.Context, r s3Request) (*http.Response, error) {
	creds, err := c.credentials.retrieve(ctx)
	if err != nil {
		return nil, err
	}
	t := c.now().UTC()
	u := c.objectURL(r.bucket, r.key)
	u.RawQuery = canonicalQuery(r.query)
	headers := map[string]string{}
	for k, v := range r.headers {
		headers[strings.ToLower(k)] = v
	}
	if r.checksum == checksumRequired || (r.checksum == checksumSupported && !c.checksumWhenRequired) {
		headers["x-amz-checksum-crc32"] = crc32Base64(r.body)
		if !c.checksumWhenRequired {
			headers["x-amz-sdk-checksum-algorithm"] = "CRC32"
		}
	}
	headers["x-amz-content-sha256"] = sha256Hex(r.body)
	headers["x-amz-date"] = t.Format("20060102T150405Z")
	if creds.SessionToken != "" {
		headers["x-amz-security-token"] = creds.SessionToken
	}
	if len(r.body) > 0 || r.method == http.MethodPut || r.method == http.MethodPost {
		headers["content-length"] = strconv.Itoa(len(r.body))
	}
	headers["host"] = u.Host

	names := make([]string, 0, len(headers))
	for k := range headers {
		if k == "cache-control" {
			continue
		}
		names = append(names, k)
	}
	sort.Strings(names)
	var ch strings.Builder
	for _, k := range names {
		ch.WriteString(k + ":" + strings.TrimSpace(headers[k]) + "\n")
	}
	signed := strings.Join(names, ";")
	canonical := strings.Join([]string{r.method, u.EscapedPath(), u.RawQuery, ch.String(), signed, headers["x-amz-content-sha256"]}, "\n")
	sig := c.signature(creds, t, canonical)
	headers["authorization"] = fmt.Sprintf("%s Credential=%s/%s/%s/s3/aws4_request, SignedHeaders=%s, Signature=%s",
		sigV4Algorithm, creds.AccessKeyID, t.Format("20060102"), c.region, signed, sig)

	req, err := http.NewRequestWithContext(ctx, r.method, u.String(), bytes.NewReader(r.body))
	if err != nil {
		return nil, err
	}
	req.ContentLength = int64(len(r.body))
	if len(r.body) == 0 {
		req.Body = http.NoBody
	}
	for k, v := range headers {
		switch k {
		case "host", "content-length":
			continue
		}
		req.Header.Set(k, v)
	}
	return c.httpClient.Do(req)
}

// parseS3Error reads an S3 error response (<Error><Code/><Message/>).
func parseS3Error(res *http.Response) error {
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	var body struct {
		Code    string `xml:"Code"`
		Message string `xml:"Message"`
	}
	_ = xml.Unmarshal(raw, &body)
	e := &S3Error{Code: body.Code, Message: body.Message, StatusCode: res.StatusCode}
	if e.Code == "" {
		switch res.StatusCode {
		case http.StatusNotFound:
			e.Code = "NotFound"
		default:
			e.Code = "Unknown"
		}
	}
	if e.Message == "" {
		e.Message = e.Code
		if e.Code == "Unknown" || e.Code == "NotFound" {
			e.Message = "UnknownError"
		}
	}
	return e
}

func drain(res *http.Response) {
	_, _ = io.Copy(io.Discard, res.Body)
	res.Body.Close()
}

// putObjectInput mirrors the PutObjectCommand parameters the provider sets.
type putObjectInput struct {
	Bucket       string
	Key          string
	ACL          string
	ContentType  string
	CacheControl string
	Metadata     [][2]string
}

func (in putObjectInput) headers() map[string]string {
	h := map[string]string{}
	if in.ACL != "" {
		h["x-amz-acl"] = in.ACL
	}
	if in.ContentType != "" {
		h["content-type"] = in.ContentType
	}
	if in.CacheControl != "" {
		h["cache-control"] = in.CacheControl
	}
	for _, m := range in.Metadata {
		h["x-amz-meta-"+strings.ToLower(m[0])] = m[1]
	}
	return h
}

func (c *s3Client) putObject(ctx context.Context, in putObjectInput, body []byte) error {
	res, err := c.do(ctx, s3Request{method: http.MethodPut, bucket: in.Bucket, key: in.Key, query: []queryParam{{"x-id", "PutObject"}},
		headers: in.headers(), body: body, checksum: checksumSupported})
	if err != nil {
		return err
	}
	drain(res)
	return nil
}

func (c *s3Client) getObject(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	res, err := c.do(ctx, s3Request{method: http.MethodGet, bucket: bucket, key: key, query: []queryParam{{"x-id", "GetObject"}},
		headers: c.checksumModeHeaders()})
	if err != nil {
		return nil, err
	}
	return res.Body, nil
}

func (c *s3Client) deleteObject(ctx context.Context, bucket, key string) error {
	res, err := c.do(ctx, s3Request{method: http.MethodDelete, bucket: bucket, key: key, query: []queryParam{{"x-id", "DeleteObject"}}})
	if err != nil {
		return err
	}
	drain(res)
	return nil
}

func xmlEscape(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func (c *s3Client) deleteObjects(ctx context.Context, bucket string, keys []string, quiet bool) error {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><Delete xmlns="` + s3XMLNamespace + `">`)
	for _, k := range keys {
		b.WriteString("<Object><Key>" + xmlEscape(k) + "</Key></Object>")
	}
	if quiet {
		b.WriteString("<Quiet>true</Quiet>")
	}
	b.WriteString("</Delete>")
	res, err := c.do(ctx, s3Request{method: http.MethodPost, bucket: bucket, query: []queryParam{{"delete", ""}},
		headers: map[string]string{"content-type": "application/xml"}, body: []byte(b.String()), checksum: checksumRequired})
	if err != nil {
		return err
	}
	drain(res)
	return nil
}

// Multipart upload (lib-storage Upload for bodies larger than one part).

func (c *s3Client) createMultipartUpload(ctx context.Context, in putObjectInput) (string, error) {
	h := in.headers()
	if !c.checksumWhenRequired {
		h["x-amz-checksum-algorithm"] = "CRC32"
	}
	res, err := c.do(ctx, s3Request{method: http.MethodPost, bucket: in.Bucket, key: in.Key,
		query: []queryParam{{"uploads", ""}, {"x-id", "CreateMultipartUpload"}}, headers: h})
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	var out struct {
		UploadID string `xml:"UploadId"`
	}
	if err := xml.NewDecoder(res.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.UploadID, nil
}

type completedPart struct {
	ETag     string
	Checksum string
	Number   int
}

func (c *s3Client) uploadPart(ctx context.Context, bucket, key, uploadID string, number int, body []byte) (completedPart, error) {
	res, err := c.do(ctx, s3Request{method: http.MethodPut, bucket: bucket, key: key,
		query:   []queryParam{{"partNumber", strconv.Itoa(number)}, {"uploadId", uploadID}, {"x-id", "UploadPart"}},
		body:    body,
		headers: map[string]string{}, checksum: checksumSupported})
	if err != nil {
		return completedPart{}, err
	}
	drain(res)
	return completedPart{ETag: res.Header.Get("ETag"), Checksum: crc32Base64(body), Number: number}, nil
}

func (c *s3Client) completeMultipartUpload(ctx context.Context, bucket, key, uploadID string, parts []completedPart) error {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><CompleteMultipartUpload xmlns="` + s3XMLNamespace + `">`)
	for _, p := range parts {
		fmt.Fprintf(&b, "<Part><ChecksumCRC32>%s</ChecksumCRC32><ETag>%s</ETag><PartNumber>%d</PartNumber></Part>", p.Checksum, xmlEscape(p.ETag), p.Number)
	}
	b.WriteString("</CompleteMultipartUpload>")
	res, err := c.do(ctx, s3Request{method: http.MethodPost, bucket: bucket, key: key,
		query:   []queryParam{{"uploadId", uploadID}, {"x-id", "CompleteMultipartUpload"}},
		headers: map[string]string{"content-type": "application/xml"}, body: []byte(b.String())})
	if err != nil {
		return err
	}
	defer res.Body.Close()
	// S3 may answer 200 with an error document.
	raw, _ := io.ReadAll(res.Body)
	if bytes.Contains(raw, []byte("<Error>")) {
		var e struct {
			Code    string `xml:"Code"`
			Message string `xml:"Message"`
		}
		_ = xml.Unmarshal(raw, &e)
		return &S3Error{Code: e.Code, Message: e.Message, StatusCode: res.StatusCode}
	}
	return nil
}

func (c *s3Client) abortMultipartUpload(ctx context.Context, bucket, key, uploadID string) {
	res, err := c.do(ctx, s3Request{method: http.MethodDelete, bucket: bucket, key: key,
		query: []queryParam{{"uploadId", uploadID}, {"x-id", "AbortMultipartUpload"}}})
	if err == nil {
		drain(res)
	}
}

// uploadWriter mirrors lib-storage Upload over a PassThrough: data is
// buffered in parts of 5 MiB; a body that fits in one part is sent with
// PutObject, a larger one as a multipart upload. Close ends the stream and
// waits for the upload (the promise).
type uploadWriter struct {
	ctx      context.Context
	client   *s3Client
	input    putObjectInput
	buf      []byte
	uploadID string
	parts    []completedPart
	err      error
	closed   bool
}

func (w *uploadWriter) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	if w.closed {
		return 0, errors.New("write after end")
	}
	w.buf = append(w.buf, p...)
	for len(w.buf) > minPartSizeBytes {
		if err := w.flushPart(w.buf[:minPartSizeBytes]); err != nil {
			w.err = err
			return 0, err
		}
		w.buf = append([]byte(nil), w.buf[minPartSizeBytes:]...)
	}
	return len(p), nil
}

func (w *uploadWriter) WriteString(s string) (int, error) { return w.Write([]byte(s)) }

func (w *uploadWriter) flushPart(data []byte) error {
	if w.uploadID == "" {
		id, err := w.client.createMultipartUpload(w.ctx, w.input)
		if err != nil {
			return err
		}
		w.uploadID = id
	}
	part, err := w.client.uploadPart(w.ctx, w.input.Bucket, w.input.Key, w.uploadID, len(w.parts)+1, append([]byte(nil), data...))
	if err != nil {
		w.client.abortMultipartUpload(w.ctx, w.input.Bucket, w.input.Key, w.uploadID)
		return err
	}
	w.parts = append(w.parts, part)
	return nil
}

func (w *uploadWriter) Close() error {
	if w.closed {
		return w.err
	}
	w.closed = true
	if w.err != nil {
		return w.err
	}
	if w.uploadID == "" {
		w.err = w.client.putObject(w.ctx, w.input, w.buf)
		return w.err
	}
	if len(w.buf) > 0 {
		if err := w.flushPart(w.buf); err != nil {
			w.err = err
			return err
		}
	}
	if err := w.client.completeMultipartUpload(w.ctx, w.input.Bucket, w.input.Key, w.uploadID, w.parts); err != nil {
		w.client.abortMultipartUpload(w.ctx, w.input.Bucket, w.input.Key, w.uploadID)
		w.err = err
	}
	return w.err
}

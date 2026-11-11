// Package services mirrors @medusajs/file-s3/src/services.
package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	filetypes "github.com/hitkey-io/medusago/packages/core/types/src/file"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
)

var _ filetypes.IFileProvider = (*S3FileService)(nil)

// Identifier is S3FileService.identifier.
const Identifier = "s3"

const defaultUploadExpirationDurationSeconds = 60 * 60

// newULID generates the unique part of the file keys (ulid()).
var newULID = common.ULID

// s3FileServiceConfig mirrors S3FileServiceConfig.
type s3FileServiceConfig struct {
	fileURL              string
	accessKeyID          string
	secretAccessKey      string
	sessionToken         string
	authenticationMethod string
	region               string
	bucket               string
	prefix               string
	endpoint             string
	cacheControl         string
	downloadFileDuration int
	additionalConfig     map[string]any
	// acl: "" = unset (undefined), "-" = disabled (false), else the ACL.
	acl string
}

// S3FileService mirrors S3FileService.
type S3FileService struct {
	config s3FileServiceConfig
	logger *slog.Logger
	client *s3Client
}

func stringOption(options map[string]any, key string) string {
	if v, ok := options[key].(string); ok {
		return v
	}
	return ""
}

func intOption(options map[string]any, key string, def int) int {
	switch v := options[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return def
}

// NewS3FileService mirrors the constructor (options file_url,
// access_key_id, secret_access_key, session_token, authentication_method,
// region, bucket, prefix, endpoint, cache_control, download_file_duration,
// additional_client_config, acl).
func NewS3FileService(options map[string]any, logger *slog.Logger) (*S3FileService, error) {
	if logger == nil {
		logger = slog.Default()
	}
	authenticationMethod := stringOption(options, "authentication_method")
	if authenticationMethod == "" {
		authenticationMethod = "access-key"
	}
	if authenticationMethod == "access-key" && (stringOption(options, "access_key_id") == "" || stringOption(options, "secret_access_key") == "") {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Access key ID and secret access key are required when using access key authentication")
	}
	cfg := s3FileServiceConfig{
		fileURL:              stringOption(options, "file_url"),
		accessKeyID:          stringOption(options, "access_key_id"),
		secretAccessKey:      stringOption(options, "secret_access_key"),
		sessionToken:         stringOption(options, "session_token"),
		authenticationMethod: authenticationMethod,
		region:               stringOption(options, "region"),
		bucket:               stringOption(options, "bucket"),
		prefix:               stringOption(options, "prefix"),
		endpoint:             stringOption(options, "endpoint"),
		cacheControl:         "public, max-age=31536000",
		downloadFileDuration: intOption(options, "download_file_duration", 60*60),
		additionalConfig:     map[string]any{},
	}
	if v, ok := options["cache_control"].(string); ok {
		cfg.cacheControl = v
	}
	if v, ok := options["additional_client_config"].(map[string]any); ok {
		cfg.additionalConfig = v
	}
	switch v := options["acl"].(type) {
	case bool:
		if !v {
			cfg.acl = "-"
		}
	case string:
		cfg.acl = v
	}
	s := &S3FileService{config: cfg, logger: logger}
	client, err := s.getClient()
	if err != nil {
		return nil, err
	}
	s.client = client
	return s, nil
}

// getClient mirrors getClient: { credentials, region, endpoint,
// ...additional_client_config } (forcePathStyle, region, endpoint,
// credentials and maxAttempts of the client configuration are supported).
func (s *S3FileService) getClient() (*s3Client, error) {
	var creds credentialsProvider
	if s.config.authenticationMethod == "access-key" {
		creds = staticCredentials{AccessKeyID: s.config.accessKeyID, SecretAccessKey: s.config.secretAccessKey, SessionToken: s.config.sessionToken}
	} else {
		creds = newDefaultCredentialsChain()
	}
	c := &s3Client{
		region:      s.config.region,
		credentials: creds,
		maxAttempts: defaultMaxTries,
		httpClient:  http.DefaultClient,
		now:         time.Now,
	}
	endpoint := s.config.endpoint
	extra := s.config.additionalConfig
	if v, ok := extra["region"].(string); ok {
		c.region = v
	}
	if v, ok := extra["endpoint"].(string); ok {
		endpoint = v
	}
	if v, ok := extra["forcePathStyle"].(bool); ok {
		c.forcePathStyle = v
	}
	c.checksumWhenRequired = checksumSetting(extra, "requestChecksumCalculation", "AWS_REQUEST_CHECKSUM_CALCULATION")
	c.validationWhenRequired = checksumSetting(extra, "responseChecksumValidation", "AWS_RESPONSE_CHECKSUM_VALIDATION")
	if v := intOption(extra, "maxAttempts", 0); v > 0 {
		c.maxAttempts = v
	}
	if v, ok := extra["credentials"].(map[string]any); ok {
		creds := staticCredentials{AccessKeyID: stringOption(v, "accessKeyId"), SecretAccessKey: stringOption(v, "secretAccessKey"), SessionToken: stringOption(v, "sessionToken")}
		c.credentials = creds
	}
	if endpoint != "" {
		u, err := url.Parse(endpoint)
		if err != nil || u.Host == "" {
			return nil, fmt.Errorf("Invalid endpoint: %s", endpoint)
		}
		c.endpoint = u
	}
	return c, nil
}

// checksumSetting reads requestChecksumCalculation /
// responseChecksumValidation (client configuration, then environment):
// true for "WHEN_REQUIRED".
func checksumSetting(extra map[string]any, key, env string) bool {
	v, _ := extra[key].(string)
	if v == "" {
		v = os.Getenv(env)
	}
	return strings.EqualFold(v, "WHEN_REQUIRED")
}

func queryParams(m map[string]string) []queryParam {
	out := []queryParam{}
	for k, v := range m {
		out = append(out, queryParam{k, v})
	}
	return out
}

// resolveACL mirrors resolveAcl ("" = no ACL header).
func (s *S3FileService) resolveACL(access string) string {
	if s.config.acl == "-" {
		return ""
	}
	if s.config.acl != "" {
		return s.config.acl
	}
	if access == "public" {
		return "public-read"
	}
	return "private"
}

// fileKey builds `${prefix}${dir/}${name}-${ulid()}${ext}`.
func (s *S3FileService) fileKey(sanitized string) string {
	parsed := posixParse(sanitized)
	dir := ""
	if parsed.Dir != "" {
		dir = parsed.Dir + "/"
	}
	return fmt.Sprintf("%s%s%s-%s%s", s.config.prefix, dir, parsed.Name, newULID(), parsed.Ext)
}

func (s *S3FileService) fileURL(fileKey string) string {
	segments := strings.Split(fileKey, "/")
	for i, seg := range segments {
		segments[i] = encodeURIComponent(seg)
	}
	return s.config.fileURL + "/" + strings.Join(segments, "/")
}

// decodeFileContent mirrors decodeFileContent.
func decodeFileContent(content, mimeType string) []byte {
	if decoded, err := base64.StdEncoding.DecodeString(content); err == nil && base64.StdEncoding.EncodeToString(decoded) == content {
		return decoded
	}
	if strings.HasPrefix(mimeType, "text/") || strings.Contains(mimeType, "csv") || strings.Contains(mimeType, "json") || strings.Contains(mimeType, "xml") {
		return []byte(content)
	}
	// Buffer.from(content, "binary"): latin1, one byte per UTF-16 unit.
	out := make([]byte, 0, len(content))
	for _, r := range content {
		out = append(out, byte(r))
	}
	return out
}

func invalid(msg string) error {
	return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, msg)
}

// Upload mirrors upload.
func (s *S3FileService) Upload(ctx context.Context, file *filetypes.ProviderUploadFileDTO) (filetypes.ProviderFileResultDTO, error) {
	if file == nil {
		return filetypes.ProviderFileResultDTO{}, invalid("No file provided")
	}
	if file.Filename == "" {
		return filetypes.ProviderFileResultDTO{}, invalid("No filename provided")
	}
	sanitized := sanitizeFilePath(file.Filename)
	if sanitized == "" {
		return filetypes.ProviderFileResultDTO{}, invalid("Invalid filename: " + file.Filename)
	}
	key := s.fileKey(sanitized)
	content := decodeFileContent(file.Content, file.MimeType)
	in := putObjectInput{
		Bucket:       s.config.bucket,
		Key:          key,
		ACL:          s.resolveACL(file.Access),
		ContentType:  file.MimeType,
		CacheControl: s.config.cacheControl,
		Metadata:     [][2]string{{"original-filename", encodeURIComponent(file.Filename)}},
	}
	if err := s.client.putObject(ctx, in, content); err != nil {
		s.logger.Error(err.Error())
		return filetypes.ProviderFileResultDTO{}, err
	}
	return filetypes.ProviderFileResultDTO{URL: s.fileURL(key), Key: key}, nil
}

// GetUploadStream mirrors getUploadStream (lib-storage Upload over a
// PassThrough).
func (s *S3FileService) GetUploadStream(ctx context.Context, data filetypes.ProviderUploadStreamDTO) (*filetypes.ProviderUploadStream, error) {
	if data.Filename == "" {
		return nil, invalid("No filename provided")
	}
	sanitized := sanitizeFilePath(data.Filename)
	if sanitized == "" {
		return nil, invalid("Invalid filename: " + data.Filename)
	}
	key := s.fileKey(sanitized)
	w := &uploadWriter{ctx: context.WithoutCancel(ctx), client: s.client, input: putObjectInput{
		Bucket:       s.config.bucket,
		Key:          key,
		ACL:          s.resolveACL(data.Access),
		ContentType:  data.MimeType,
		CacheControl: s.config.cacheControl,
		Metadata:     [][2]string{{"original-filename", encodeURIComponent(data.Filename)}},
	}}
	return &filetypes.ProviderUploadStream{File: w, FileKey: key, URL: s.fileURL(key)}, nil
}

// Delete mirrors delete with a list (DeleteObjects, quiet). Errors are
// logged, not returned.
func (s *S3FileService) Delete(ctx context.Context, fileKeys []string) error {
	if err := s.client.deleteObjects(ctx, s.config.bucket, fileKeys, true); err != nil {
		s.logger.Error(err.Error())
	}
	return nil
}

// DeleteOne mirrors delete with a single file (DeleteObject).
func (s *S3FileService) DeleteOne(ctx context.Context, fileKey string) error {
	if err := s.client.deleteObject(ctx, s.config.bucket, fileKey); err != nil {
		s.logger.Error(err.Error())
	}
	return nil
}

// GetPresignedDownloadURL mirrors getPresignedDownloadUrl.
func (s *S3FileService) GetPresignedDownloadURL(ctx context.Context, fileKey string) (string, error) {
	return s.client.presign(ctx, http.MethodGet, s.config.bucket, fileKey,
		append(queryParams(s.client.checksumModeHeaders()), queryParam{"x-id", "GetObject"}), s.config.downloadFileDuration)
}

// GetPresignedUploadURL mirrors getPresignedUploadUrl: the ACL is only set
// when an access is given.
func (s *S3FileService) GetPresignedUploadURL(ctx context.Context, data filetypes.ProviderGetPresignedUploadUrlDTO) (filetypes.ProviderFileResultDTO, error) {
	if data.Filename == "" {
		return filetypes.ProviderFileResultDTO{}, invalid("No filename provided")
	}
	sanitized := sanitizeFilePath(data.Filename)
	if sanitized == "" {
		return filetypes.ProviderFileResultDTO{}, invalid("Invalid filename: " + data.Filename)
	}
	key := s.config.prefix + sanitized
	params := []queryParam{}
	if data.Access != "" {
		if acl := s.resolveACL(data.Access); acl != "" {
			params = append(params, queryParam{"x-amz-acl", acl})
		}
	}
	if !s.client.checksumWhenRequired {
		params = append(params, queryParam{"x-amz-checksum-crc32", crc32Base64(nil)}, queryParam{"x-amz-sdk-checksum-algorithm", "CRC32"})
	}
	params = append(params, queryParam{"x-id", "PutObject"})
	expiresIn := defaultUploadExpirationDurationSeconds
	if data.ExpiresIn != nil {
		expiresIn = *data.ExpiresIn
	}
	u, err := s.client.presign(ctx, http.MethodPut, s.config.bucket, key, params, expiresIn)
	if err != nil {
		return filetypes.ProviderFileResultDTO{}, err
	}
	return filetypes.ProviderFileResultDTO{URL: u, Key: key}, nil
}

// GetDownloadStream mirrors getDownloadStream.
func (s *S3FileService) GetDownloadStream(ctx context.Context, fileKey string) (io.ReadCloser, error) {
	if fileKey == "" {
		return nil, invalid("No fileKey provided")
	}
	return s.client.getObject(ctx, s.config.bucket, fileKey)
}

// GetAsBuffer mirrors getAsBuffer.
func (s *S3FileService) GetAsBuffer(ctx context.Context, fileKey string) ([]byte, error) {
	if fileKey == "" {
		return nil, invalid("No fileKey provided")
	}
	body, err := s.client.getObject(ctx, s.config.bucket, fileKey)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	return io.ReadAll(body)
}

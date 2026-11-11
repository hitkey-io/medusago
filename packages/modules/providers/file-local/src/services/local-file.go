// Package services mirrors @medusajs/file-local/src/services.
package services

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	filetypes "github.com/hitkey-io/medusago/packages/core/types/src/file"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
)

var _ filetypes.IFileProvider = (*LocalFileService)(nil)

// LocalFileService mirrors LocalFileService.
type LocalFileService struct {
	uploadDir        string
	privateUploadDir string
	backendURL       string
}

// NewLocalFileService mirrors the constructor (options upload_dir,
// private_upload_dir, backend_url).
func NewLocalFileService(options map[string]any) *LocalFileService {
	cwd, _ := os.Getwd()
	s := &LocalFileService{
		uploadDir:        filepath.Join(cwd, "static"),
		privateUploadDir: filepath.Join(cwd, "static"),
		backendURL:       "http://localhost:9000/static",
	}
	if v, ok := options["upload_dir"].(string); ok && v != "" {
		s.uploadDir = v
	}
	if v, ok := options["private_upload_dir"].(string); ok && v != "" {
		s.privateUploadDir = v
	}
	if v, ok := options["backend_url"].(string); ok && v != "" {
		s.backendURL = v
	}
	return s
}

// UploadFile mirrors ProviderUploadFileDTO.
type UploadFile = filetypes.ProviderUploadFileDTO

// FileResult mirrors ProviderFileResultDTO.
type FileResult = filetypes.ProviderFileResultDTO

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

func (s *LocalFileService) baseDir(access string) string {
	if access == "public" {
		return s.uploadDir
	}
	return s.privateUploadDir
}

func (s *LocalFileService) keyDir(fileKey string) string {
	if strings.HasPrefix(fileKey, "private-") {
		return s.privateUploadDir
	}
	return s.uploadDir
}

// filePath mirrors getUploadFilePath (keys may not leave the directory).
func filePath(baseDir, fileKey string) (string, error) {
	base, _ := filepath.Abs(baseDir)
	resolved := filepath.Join(base, fileKey)
	if filepath.IsAbs(fileKey) {
		resolved = filepath.Clean(fileKey)
	}
	rel, err := filepath.Rel(base, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Invalid file key: %s", fileKey))
	}
	return resolved, nil
}

// fileURL mirrors getUploadFileUrl.
func (s *LocalFileService) fileURL(fileKey string) string {
	u, err := url.Parse(s.backendURL)
	if err != nil {
		return s.backendURL + "/" + fileKey
	}
	u.Path = path.Join(u.Path, fileKey)
	return u.String()
}

func (s *LocalFileService) newKey(filename, access string) (string, string) {
	dir, base := path.Split(filename)
	prefix := "private-"
	if access == "public" {
		prefix = ""
	}
	return path.Join(dir, fmt.Sprintf("%s%d-%s", prefix, time.Now().UnixMilli(), base)), dir
}

// Upload mirrors upload.
func (s *LocalFileService) Upload(ctx context.Context, file *UploadFile) (FileResult, error) {
	if file == nil {
		return FileResult{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "No file provided")
	}
	if file.Filename == "" {
		return FileResult{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "No filename provided")
	}
	baseDir := s.baseDir(file.Access)
	key, dir := s.newKey(file.Filename, file.Access)
	if err := os.MkdirAll(filepath.Join(baseDir, dir), 0o755); err != nil {
		return FileResult{}, err
	}
	p, err := filePath(baseDir, key)
	if err != nil {
		return FileResult{}, err
	}
	if err := os.WriteFile(p, decodeFileContent(file.Content, file.MimeType), 0o644); err != nil {
		return FileResult{}, err
	}
	return FileResult{Key: key, URL: s.fileURL(key)}, nil
}

// Delete mirrors delete (missing files are ignored).
func (s *LocalFileService) Delete(ctx context.Context, fileKeys []string) error {
	for _, key := range fileKeys {
		p, err := filePath(s.keyDir(key), key)
		if err != nil {
			return err
		}
		if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return nil
}

// GetAsBuffer mirrors getAsBuffer.
func (s *LocalFileService) GetAsBuffer(ctx context.Context, fileKey string) ([]byte, error) {
	p, err := filePath(s.keyDir(fileKey), fileKey)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(p)
}

// GetDownloadStream mirrors getDownloadStream.
func (s *LocalFileService) GetDownloadStream(ctx context.Context, fileKey string) (io.ReadCloser, error) {
	p, err := filePath(s.keyDir(fileKey), fileKey)
	if err != nil {
		return nil, err
	}
	return os.Open(p)
}

// GetPresignedDownloadURL mirrors getPresignedDownloadUrl.
func (s *LocalFileService) GetPresignedDownloadURL(ctx context.Context, fileKey string) (string, error) {
	p, err := filePath(s.keyDir(fileKey), fileKey)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(p); err != nil {
		return "", common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("File with key %s not found", fileKey))
	}
	return s.fileURL(fileKey), nil
}

// GetPresignedUploadURL mirrors getPresignedUploadUrl.
func (s *LocalFileService) GetPresignedUploadURL(ctx context.Context, data filetypes.ProviderGetPresignedUploadUrlDTO) (FileResult, error) {
	if data.Filename == "" {
		return FileResult{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "No filename provided")
	}
	return FileResult{URL: "/admin/uploads", Key: data.Filename}, nil
}

// UploadDir is the public upload directory (served under /static).
func (s *LocalFileService) UploadDir() string { return s.uploadDir }

// UploadStream mirrors the result of getUploadStream.
type UploadStream = filetypes.ProviderUploadStream

// GetUploadStream mirrors getUploadStream: the file is created at once and
// written through the returned writer; Close finishes it (the stream's
// promise).
func (s *LocalFileService) GetUploadStream(ctx context.Context, data filetypes.ProviderUploadStreamDTO) (*UploadStream, error) {
	if data.Filename == "" {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "No filename provided")
	}
	baseDir := s.baseDir(data.Access)
	key, dir := s.newKey(data.Filename, data.Access)
	if err := os.MkdirAll(filepath.Join(baseDir, dir), 0o755); err != nil {
		return nil, err
	}
	p, err := filePath(baseDir, key)
	if err != nil {
		return nil, err
	}
	f, err := os.Create(p)
	if err != nil {
		return nil, err
	}
	return &UploadStream{File: f, FileKey: key, URL: s.fileURL(key)}, nil
}

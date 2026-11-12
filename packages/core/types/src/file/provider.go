// Package file mirrors @medusajs/types/src/file.
package file

import (
	"context"
	"io"
)

// ProviderUploadFileDTO mirrors ProviderUploadFileDTO.
type ProviderUploadFileDTO struct {
	Filename string
	MimeType string
	// Content is base64, UTF-8 text or a binary (latin1) string.
	Content string
	// Access is "public" or "private" ("" = unset).
	Access string
}

// ProviderFileResultDTO mirrors ProviderFileResultDTO.
type ProviderFileResultDTO struct {
	URL string `json:"url"`
	Key string `json:"key"`
}

// ProviderGetPresignedUploadUrlDTO mirrors ProviderGetPresignedUploadUrlDTO.
type ProviderGetPresignedUploadUrlDTO struct {
	Filename string
	MimeType string
	Access   string
	// ExpiresIn is the validity of the URL in seconds (nil = provider
	// default).
	ExpiresIn *int
}

// ProviderUploadStreamDTO mirrors ProviderUploadStreamDTO.
type ProviderUploadStreamDTO struct {
	Filename string
	MimeType string
	Access   string
}

// UploadWriter is the writable side of an upload stream. Close ends the
// stream and waits for the upload (the stream's promise).
type UploadWriter interface {
	io.Writer
	io.StringWriter
	io.Closer
}

// ProviderUploadStream mirrors the result of getUploadStream
// ({ writeStream, promise, url, fileKey }).
type ProviderUploadStream struct {
	File    UploadWriter
	FileKey string
	URL     string
}

// IFileProvider mirrors IFileProvider (AbstractFileProviderService).
type IFileProvider interface {
	Upload(ctx context.Context, file *ProviderUploadFileDTO) (ProviderFileResultDTO, error)
	// Delete deletes files: the module always passes a list.
	Delete(ctx context.Context, fileKeys []string) error
	GetPresignedDownloadURL(ctx context.Context, fileKey string) (string, error)
	GetPresignedUploadURL(ctx context.Context, data ProviderGetPresignedUploadUrlDTO) (ProviderFileResultDTO, error)
	GetUploadStream(ctx context.Context, data ProviderUploadStreamDTO) (*ProviderUploadStream, error)
	GetDownloadStream(ctx context.Context, fileKey string) (io.ReadCloser, error)
	GetAsBuffer(ctx context.Context, fileKey string) ([]byte, error)
}

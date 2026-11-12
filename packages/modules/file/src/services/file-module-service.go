// Package services mirrors @medusajs/file/src/services.
package services

import (
	"context"
	"fmt"
	"io"

	filetypes "github.com/hitkey-io/medusago/packages/core/types/src/file"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// FileModuleService mirrors FileModuleService over the (single) file
// provider.
type FileModuleService struct {
	provider filetypes.IFileProvider
}

// NewFileModuleService builds the service.
func NewFileModuleService(provider filetypes.IFileProvider) *FileModuleService {
	return &FileModuleService{provider: provider}
}

// Provider returns the file provider.
func (s *FileModuleService) Provider() filetypes.IFileProvider { return s.provider }

func fileRecord(id, url any) *dal.Record {
	r := dal.NewRecord()
	r.Set("id", id)
	r.Set("url", url)
	return r
}

// CreateFiles mirrors createFiles.
func (s *FileModuleService) CreateFiles(ctx context.Context, files []*filetypes.ProviderUploadFileDTO) ([]*dal.Record, error) {
	out := make([]*dal.Record, len(files))
	for i, f := range files {
		res, err := s.provider.Upload(ctx, f)
		if err != nil {
			return nil, err
		}
		out[i] = fileRecord(res.Key, res.URL)
	}
	return out, nil
}

// DeleteFiles mirrors deleteFiles.
func (s *FileModuleService) DeleteFiles(ctx context.Context, ids []string) error {
	return s.provider.Delete(ctx, ids)
}

// RetrieveFile mirrors retrieveFile.
func (s *FileModuleService) RetrieveFile(ctx context.Context, id string) (*dal.Record, error) {
	u, err := s.provider.GetPresignedDownloadURL(ctx, id)
	if err != nil {
		return nil, err
	}
	return fileRecord(id, u), nil
}

// GetUploadFileURL mirrors getUploadFileUrls for one file (FileProviderService
// getPresignedUploadUrl).
func (s *FileModuleService) GetUploadFileURL(ctx context.Context, data filetypes.ProviderGetPresignedUploadUrlDTO) (filetypes.ProviderFileResultDTO, error) {
	if data.Filename == "" {
		return filetypes.ProviderFileResultDTO{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "File name is required to get a presigned upload URL")
	}
	return s.provider.GetPresignedUploadURL(ctx, data)
}

func (s *FileModuleService) listFiles(ctx context.Context, filters *dal.Record) ([]*dal.Record, error) {
	var id any
	if filters != nil {
		id, _ = filters.Get("id")
	}
	if id == nil || id == "" {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Listing of files is only supported when filtering by ID.")
	}
	ids, isList := id.([]any)
	if !isList {
		ids = []any{id}
	}
	out := make([]*dal.Record, 0, len(ids))
	for _, x := range ids {
		key := fmt.Sprint(x)
		u, err := s.provider.GetPresignedDownloadURL(ctx, key)
		if err != nil {
			return nil, err
		}
		out = append(out, fileRecord(x, u))
	}
	return out, nil
}

// ListBySuffix mirrors listFiles (the only list method).
func (s *FileModuleService) ListBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error) {
	return s.listFiles(ctx, filters)
}

// ListAndCountBySuffix mirrors listAndCountFiles.
func (s *FileModuleService) ListAndCountBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, int, error) {
	rows, err := s.listFiles(ctx, filters)
	if err != nil {
		return nil, 0, err
	}
	return rows, len(rows), nil
}

// GetUploadStream mirrors getUploadStream.
func (s *FileModuleService) GetUploadStream(ctx context.Context, data filetypes.ProviderUploadStreamDTO) (*filetypes.ProviderUploadStream, error) {
	return s.provider.GetUploadStream(ctx, data)
}

// GetDownloadStream mirrors getDownloadStream.
func (s *FileModuleService) GetDownloadStream(ctx context.Context, id string) (io.ReadCloser, error) {
	return s.provider.GetDownloadStream(ctx, id)
}

// GetAsBuffer mirrors getAsBuffer.
func (s *FileModuleService) GetAsBuffer(ctx context.Context, id string) ([]byte, error) {
	return s.provider.GetAsBuffer(ctx, id)
}

// GetProvider mirrors getProvider.
func (s *FileModuleService) GetProvider() filetypes.IFileProvider { return s.provider }

// GetUploadFileUrls mirrors getUploadFileUrls (array input): a presigned
// upload URL per file.
func (s *FileModuleService) GetUploadFileUrls(ctx context.Context, data []filetypes.ProviderGetPresignedUploadUrlDTO) ([]filetypes.ProviderFileResultDTO, error) {
	out := make([]filetypes.ProviderFileResultDTO, 0, len(data))
	for _, d := range data {
		r, err := s.provider.GetPresignedUploadURL(ctx, d)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

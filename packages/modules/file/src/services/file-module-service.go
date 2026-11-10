// Package services mirrors @medusajs/file/src/services.
package services

import (
	"context"
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	localfile "github.com/hitkey-io/medusago/packages/modules/providers/file-local/src/services"
)

// FileModuleService mirrors FileModuleService over the (single) file
// provider.
type FileModuleService struct {
	provider *localfile.LocalFileService
}

// NewFileModuleService builds the service.
func NewFileModuleService(provider *localfile.LocalFileService) *FileModuleService {
	return &FileModuleService{provider: provider}
}

// Provider returns the file provider.
func (s *FileModuleService) Provider() *localfile.LocalFileService { return s.provider }

func fileRecord(id, url any) *dal.Record {
	r := dal.NewRecord()
	r.Set("id", id)
	r.Set("url", url)
	return r
}

// CreateFiles mirrors createFiles.
func (s *FileModuleService) CreateFiles(ctx context.Context, files []*localfile.UploadFile) ([]*dal.Record, error) {
	out := make([]*dal.Record, len(files))
	for i, f := range files {
		res, err := s.provider.Upload(f)
		if err != nil {
			return nil, err
		}
		out[i] = fileRecord(res.Key, res.URL)
	}
	return out, nil
}

// DeleteFiles mirrors deleteFiles.
func (s *FileModuleService) DeleteFiles(ctx context.Context, ids []string) error {
	return s.provider.Delete(ids)
}

// RetrieveFile mirrors retrieveFile.
func (s *FileModuleService) RetrieveFile(ctx context.Context, id string) (*dal.Record, error) {
	u, err := s.provider.GetPresignedDownloadURL(id)
	if err != nil {
		return nil, err
	}
	return fileRecord(id, u), nil
}

// GetUploadFileURL mirrors getUploadFileUrls for one file.
func (s *FileModuleService) GetUploadFileURL(ctx context.Context, filename string) (localfile.FileResult, error) {
	return s.provider.GetPresignedUploadURL(filename)
}

func (s *FileModuleService) listFiles(filters *dal.Record) ([]*dal.Record, error) {
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
		u, err := s.provider.GetPresignedDownloadURL(key)
		if err != nil {
			return nil, err
		}
		out = append(out, fileRecord(x, u))
	}
	return out, nil
}

// ListBySuffix mirrors listFiles (the only list method).
func (s *FileModuleService) ListBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error) {
	return s.listFiles(filters)
}

// ListAndCountBySuffix mirrors listAndCountFiles.
func (s *FileModuleService) ListAndCountBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, int, error) {
	rows, err := s.listFiles(filters)
	if err != nil {
		return nil, 0, err
	}
	return rows, len(rows), nil
}

// GetUploadStream mirrors getUploadStream.
func (s *FileModuleService) GetUploadStream(ctx context.Context, filename, access string) (*localfile.UploadStream, error) {
	return s.provider.GetUploadStream(filename, access)
}

package services

import (
	"context"
	"encoding/json"
	"testing"

	filetypes "github.com/hitkey-io/medusago/packages/core/types/src/file"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	localfile "github.com/hitkey-io/medusago/packages/modules/providers/file-local/src/services"
)

func TestGetUploadFileUrls(t *testing.T) {
	s := &FileModuleService{provider: localfile.NewLocalFileService(map[string]any{"upload_dir": t.TempDir()})}
	if s.GetProvider() == nil {
		t.Fatal("GetProvider returned nil")
	}
	out, err := s.GetUploadFileUrls(context.Background(), []filetypes.ProviderGetPresignedUploadUrlDTO{
		{Filename: "a.csv", MimeType: "text/csv", Access: "private"},
		{Filename: "b.png", MimeType: "image/png", Access: "public"},
	})
	if err != nil {
		t.Fatal(err)
	}
	// the local provider answers { url: "/admin/uploads", key: filename }
	raw, _ := json.Marshal(out)
	if want := `[{"url":"/admin/uploads","key":"a.csv"},{"url":"/admin/uploads","key":"b.png"}]`; string(raw) != want {
		t.Fatalf("result = %s", raw)
	}
	_, err = s.GetUploadFileUrls(context.Background(), []filetypes.ProviderGetPresignedUploadUrlDTO{{MimeType: "text/csv"}})
	if me, ok := common.AsMedusaError(err); !ok || me.Type != common.MedusaErrorTypes.INVALID_DATA {
		t.Fatalf("err = %v", err)
	}
}

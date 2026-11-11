package steps

import (
	"context"
	"fmt"
	"strconv"

	filetypes "github.com/hitkey-io/medusago/packages/core/types/src/file"
	"github.com/hitkey-io/medusago/packages/core/utils/src/csv"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// Helpers shared by the export steps (orders, customers, inventory items,
// products).

// FileStreamService is the part of IFileModuleService the export steps use.
type FileStreamService interface {
	GetUploadStream(ctx context.Context, data filetypes.ProviderUploadStreamDTO) (*filetypes.ProviderUploadStream, error)
	DeleteFiles(ctx context.Context, ids []string) error
}

// FileStreamServiceOf resolves the file module.
func FileStreamServiceOf(c composer.Container) FileStreamService {
	s, _ := c.Resolve(utilsmodulessdk.FILE).(FileStreamService)
	return s
}

// Deduplicate mirrors deduplicate (first occurrence wins).
func Deduplicate(list []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range list {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// BatchSize mirrors !isNaN(parseInt(batch_size)) ? parseInt(batch_size) : def.
func BatchSize(v any, def int) int {
	if v == nil {
		return def
	}
	s := fmt.Sprint(v)
	digits := ""
	for i, c := range s {
		if (c >= '0' && c <= '9') || (i == 0 && (c == '-' || c == '+')) {
			digits += string(c)
			continue
		}
		break
	}
	n, err := strconv.Atoi(digits)
	if err != nil {
		return def
	}
	return n
}

// JSONRoundTrip mirrors JSON.parse(JSON.stringify(v)): records keep their
// key order, numbers become float64, dates ISO strings.
func JSONRoundTrip(v any) any {
	b, err := dal.MarshalValue(v)
	if err != nil {
		return nil
	}
	out, err := dal.DecodeOrderedJSON(b)
	if err != nil {
		return nil
	}
	return out
}

// ExportCsvOptions are the json2csv options of the export steps.
func ExportCsvOptions(prependHeader bool) csv.Json2CsvOptions {
	empty := ""
	return csv.Json2CsvOptions{
		PrependHeader: prependHeader, ArrayIndexesAsKeys: true, ExpandNestedObjects: true, ExpandArrayObjects: true,
		PreventCsvInjection: true, EmptyFieldValue: &empty,
	}
}

// FeedNotification builds a { to, channel: "feed", template: "admin-ui",
// data } notification.
func FeedNotification(to any, data *dal.Record) *dal.Record {
	n := dal.NewRecord()
	n.Set("to", to)
	n.Set("channel", "feed")
	n.Set("template", "admin-ui")
	n.Set("data", data)
	return n
}

// ExportFeedData builds the notification data of an export ({ title,
// description, file? }).
func ExportFeedData(title, description string, file *dal.Record) *dal.Record {
	d := dal.NewRecord()
	d.Set("title", title)
	d.Set("description", description)
	if file != nil {
		d.Set("file", file)
	}
	return d
}

// ExportedFile builds { filename, url, mimeType }.
func ExportedFile(filename string, url any, mimeType string) *dal.Record {
	f := dal.NewRecord()
	f.Set("filename", filename)
	f.Set("url", url)
	f.Set("mimeType", mimeType)
	return f
}

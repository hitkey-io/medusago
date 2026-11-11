package utils

import (
	"context"
	"fmt"

	searchtypes "github.com/hitkey-io/medusago/packages/core/types/src/search"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// DeleteIndexEntirely mirrors deleteIndexEntirely: every physical index
// ever built for the index is dropped and the rows behind them are hard
// deleted, so the next migration rebuilds it from version 1.
func DeleteIndexEntirely(ctx context.Context, c *Context, name string) (searchtypes.DeleteResult, error) {
	if _, err := RetrieveIndexDefinition(c.Indexes, name); err != nil {
		return searchtypes.DeleteResult{}, err
	}
	if c.Locking != nil {
		var out searchtypes.DeleteResult
		err := c.Locking.Execute(ctx, "search:seed:"+name, func(ctx context.Context) error {
			var err error
			out, err = deleteOne(ctx, c, name)
			return err
		})
		return out, err
	}
	return deleteOne(ctx, c, name)
}

func deleteOne(ctx context.Context, c *Context, name string) (searchtypes.DeleteResult, error) {
	take := 1
	records, err := ListIndexRecords(ctx, c, filter("name", name), utilsmodulessdk.FindConfig{Take: &take, WithDeleted: true})
	if err != nil {
		return searchtypes.DeleteResult{}, err
	}
	if len(records) == 0 {
		return searchtypes.DeleteResult{Index: name, DeletedVersions: 0}, nil
	}
	record := records[0]
	versions, err := listVersionsOf(ctx, c, record.ID, true)
	if err != nil {
		return searchtypes.DeleteResult{}, err
	}
	for _, version := range versions {
		err := func() error {
			provider, err := c.Providers.Retrieve(version.Provider)
			if err != nil {
				return err
			}
			_, err = provider.DeleteIndex(ctx, version.PhysicalName)
			return err
		}()
		if err != nil {
			c.Logger.Warn(fmt.Sprintf(`[Search] Cannot drop physical search index "%s": %s`, version.PhysicalName, err.Error()))
		}
	}
	versionIDs := make([]any, len(versions))
	for i, v := range versions {
		versionIDs[i] = v.ID
	}
	if len(versionIDs) > 0 {
		syncs, err := c.SyncService.List(ctx, filter("search_index_version_id", versionIDs), utilsmodulessdk.FindConfig{Select: []string{"id"}, WithDeleted: true})
		if err != nil {
			return searchtypes.DeleteResult{}, err
		}
		if len(syncs) > 0 {
			ids := make([]any, len(syncs))
			for i, s := range syncs {
				ids[i], _ = s.Get("id")
			}
			if _, err := c.SyncService.Delete(ctx, ids); err != nil {
				return searchtypes.DeleteResult{}, err
			}
		}
		if _, err := c.VersionService.Delete(ctx, versionIDs); err != nil {
			return searchtypes.DeleteResult{}, err
		}
	}
	if _, err := c.IndexService.Delete(ctx, []any{record.ID}); err != nil {
		return searchtypes.DeleteResult{}, err
	}
	if c.ActiveVersionCache != nil {
		c.ActiveVersionCache.Invalidate()
	}
	return searchtypes.DeleteResult{Index: name, DeletedVersions: len(versions)}, nil
}

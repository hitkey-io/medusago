package utils

import (
	"context"
	"fmt"

	searchtypes "github.com/hitkey-io/medusago/packages/core/types/src/search"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// VersionPhysicalName mirrors versionPhysicalName.
func VersionPhysicalName(definition *searchtypes.ResolvedDefinition, version int) string {
	return fmt.Sprintf("%s_v%d", definition.PhysicalName, version)
}

func listVersionsOf(ctx context.Context, c *Context, indexID string, withDeleted bool) ([]VersionRecord, error) {
	return ListVersionRecords(ctx, c, filter("search_index_id", indexID), utilsmodulessdk.FindConfig{WithDeleted: withDeleted})
}

// CleanupStaleVersions mirrors cleanupStaleVersions: drops every version
// below the active one.
func CleanupStaleVersions(ctx context.Context, c *Context, record IndexRecord) error {
	if record.ActiveVersion == nil {
		return nil
	}
	versions, err := listVersionsOf(ctx, c, record.ID, false)
	if err != nil {
		return err
	}
	var stale []VersionRecord
	for _, v := range versions {
		if v.Version < *record.ActiveVersion {
			stale = append(stale, v)
		}
	}
	deleteVersions(ctx, c, stale)
	return nil
}

// CleanupAbandonedVersions mirrors cleanupAbandonedVersions: drops the
// versions above the active one a run started filling and never swapped in.
func CleanupAbandonedVersions(ctx context.Context, c *Context, record IndexRecord) error {
	active := 0
	if record.ActiveVersion != nil {
		active = *record.ActiveVersion
	}
	versions, err := listVersionsOf(ctx, c, record.ID, false)
	if err != nil {
		return err
	}
	var abandoned []VersionRecord
	for _, v := range versions {
		if v.Version > active && (v.Status == StateBuilding || v.Status == StateError) {
			abandoned = append(abandoned, v)
		}
	}
	deleteVersions(ctx, c, abandoned)
	return nil
}

// CleanupAfterSwap mirrors cleanupAfterSwap (never fails).
func CleanupAfterSwap(ctx context.Context, c *Context, recordID string, activeVersion int, previousVersion *int) {
	versions, err := listVersionsOf(ctx, c, recordID, false)
	if err != nil {
		c.Logger.Warn(fmt.Sprintf("[Search] Cannot clean up after swapping in version %d: %s", activeVersion, err.Error()))
		return
	}
	var stale []VersionRecord
	for _, v := range versions {
		if v.Version < activeVersion && (previousVersion == nil || v.Version != *previousVersion) {
			stale = append(stale, v)
		}
	}
	deleteVersions(ctx, c, stale)
}

// DropIndex mirrors dropIndex: every physical index of a logical index,
// then its record (kept when a version could not be removed).
func DropIndex(ctx context.Context, c *Context, name string) error {
	records, err := ListIndexRecords(ctx, c, filter("name", name), utilsmodulessdk.FindConfig{})
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}
	record := records[0]
	versions, err := listVersionsOf(ctx, c, record.ID, false)
	if err != nil {
		return err
	}
	failed := deleteVersions(ctx, c, versions)
	if len(failed) > 0 {
		c.Logger.Warn(fmt.Sprintf(`[Search] Keeping the record for search index "%s": %d of its %d version(s) could not be removed, so the next migration plans the drop again`, name, len(failed), len(versions)))
		return nil
	}
	_, err = c.IndexService.SoftDelete(ctx, []any{record.ID})
	return err
}

// deleteVersions mirrors deleteVersions: each version's physical index,
// then its record (soft deleted); the ones that fail are returned.
func deleteVersions(ctx context.Context, c *Context, versions []VersionRecord) []VersionRecord {
	var failed []VersionRecord
	for _, version := range versions {
		err := func() error {
			provider, err := c.Providers.Retrieve(version.Provider)
			if err != nil {
				return err
			}
			if _, err := provider.DeleteIndex(ctx, version.PhysicalName); err != nil {
				return err
			}
			_, err = c.VersionService.SoftDelete(ctx, []any{version.ID})
			return err
		}()
		if err != nil {
			failed = append(failed, version)
			c.Logger.Warn(fmt.Sprintf(`[Search] Cannot delete search index "%s": %s`, version.PhysicalName, err.Error()))
		}
	}
	return failed
}

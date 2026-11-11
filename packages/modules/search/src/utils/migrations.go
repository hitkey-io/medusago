package utils

import (
	"context"

	searchtypes "github.com/hitkey-io/medusago/packages/core/types/src/search"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// CreateIndexMigrationPlan mirrors createIndexMigrationPlan.
func CreateIndexMigrationPlan(ctx context.Context, c *Context) ([]searchtypes.MigrationAction, error) {
	definitions := c.Indexes.Values()
	records, err := ListIndexRecords(ctx, c, dal.NewRecord(), utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	if len(definitions) == 0 && len(records) == 0 {
		return []searchtypes.MigrationAction{}, nil
	}
	byName := map[string]IndexRecord{}
	ids := make([]string, len(records))
	for i, r := range records {
		byName[r.Name] = r
		ids[i] = r.ID
	}
	versionsByIndexID, err := ListVersionsByIndexID(ctx, c, ids)
	if err != nil {
		return nil, err
	}
	actions := []searchtypes.MigrationAction{}
	for _, definition := range definitions {
		shared := searchtypes.MigrationAction{Index: definition.Name, DefinitionHash: definition.DefinitionHash}
		record, ok := byName[definition.Name]
		if !ok {
			a := shared
			a.Action = "create"
			a.PhysicalName = VersionPhysicalName(definition, 1)
			a.Version = 1
			actions = append(actions, a)
			continue
		}
		versions := versionsByIndexID[record.ID]
		var activeVersion *VersionRecord
		if record.ActiveVersion != nil {
			for i := range versions {
				if versions[i].Version == *record.ActiveVersion {
					activeVersion = &versions[i]
					break
				}
			}
		}
		if activeVersion != nil && activeVersion.DefinitionHash == definition.DefinitionHash && activeVersion.Provider == definition.Provider {
			a := shared
			a.Action = "noop"
			a.PhysicalName = activeVersion.PhysicalName
			actions = append(actions, a)
			continue
		}
		var pending *VersionRecord
		for i := range versions {
			v := versions[i]
			if (record.ActiveVersion == nil || v.Version > *record.ActiveVersion) && v.DefinitionHash == definition.DefinitionHash && v.Provider == definition.Provider {
				pending = &versions[i]
				break
			}
		}
		if pending != nil {
			a := shared
			a.Action = "noop"
			a.PhysicalName = pending.PhysicalName
			actions = append(actions, a)
			continue
		}
		highest := 0
		if len(versions) > 0 {
			highest = versions[0].Version
		}
		version := highest + 1
		physicalName := VersionPhysicalName(definition, version)
		if activeVersion == nil {
			a := shared
			a.Action = "create"
			a.PhysicalName = physicalName
			a.Version = version
			actions = append(actions, a)
			continue
		}
		a := shared
		a.Action = "migrate"
		a.PhysicalName = physicalName
		a.Version = version
		a.ActivePhysicalName = activeVersion.PhysicalName
		a.ActiveDefinitionHash = activeVersion.DefinitionHash
		a.Provider = definition.Provider
		if activeVersion.Provider != definition.Provider {
			a.PreviousProvider = activeVersion.Provider
		}
		actions = append(actions, a)
	}
	for _, record := range records {
		if _, declared := c.Indexes.Get(record.Name); declared {
			continue
		}
		var names []string
		for _, v := range versionsByIndexID[record.ID] {
			names = append(names, v.PhysicalName)
		}
		actions = append(actions, searchtypes.MigrationAction{Action: "drop", Index: record.Name, PhysicalNames: names})
	}
	return actions, nil
}

// ExecuteIndexMigrationPlan mirrors executeIndexMigrationPlan.
func ExecuteIndexMigrationPlan(ctx context.Context, c *Context, actions []searchtypes.MigrationAction) error {
	for _, action := range actions {
		switch action.Action {
		case "drop":
			if err := DropIndex(ctx, c, action.Index); err != nil {
				return err
			}
			continue
		case "noop":
			record, err := getOrCreateIndexRecord(ctx, c, action.Index)
			if err != nil {
				return err
			}
			if err := CleanupStaleVersions(ctx, c, record); err != nil {
				return err
			}
			continue
		}
		definition, err := RetrieveIndexDefinition(c.Indexes, action.Index)
		if err != nil {
			return err
		}
		provider, err := c.Providers.Retrieve(definition.Provider)
		if err != nil {
			return err
		}
		if _, err := provider.UpsertIndex(ctx, definition.WithPhysicalName(action.PhysicalName)); err != nil {
			return err
		}
		record, err := getOrCreateIndexRecord(ctx, c, definition.Name)
		if err != nil {
			return err
		}
		if _, err := c.VersionService.Create(ctx, []*dal.Record{filter(
			"search_index_id", record.ID,
			"version", float64(action.Version),
			"provider", definition.Provider,
			"physical_name", action.PhysicalName,
			"definition_hash", action.DefinitionHash,
			"status", StatePending,
		)}); err != nil {
			return err
		}
		if err := CleanupStaleVersions(ctx, c, record); err != nil {
			return err
		}
	}
	return nil
}

func getOrCreateIndexRecord(ctx context.Context, c *Context, name string) (IndexRecord, error) {
	existing, err := ListIndexRecords(ctx, c, filter("name", name), utilsmodulessdk.FindConfig{})
	if err != nil {
		return IndexRecord{}, err
	}
	if len(existing) > 0 {
		return existing[0], nil
	}
	created, err := c.IndexService.Create(ctx, []*dal.Record{filter("name", name, "active_version", nil)})
	if err != nil {
		return IndexRecord{}, err
	}
	return toIndexRecord(created[0]), nil
}

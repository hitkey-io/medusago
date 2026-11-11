package utils

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	searchtypes "github.com/hitkey-io/medusago/packages/core/types/src/search"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	searchutils "github.com/hitkey-io/medusago/packages/core/utils/src/search"
)

// progressLogInterval mirrors PROGRESS_LOG_INTERVAL_MS.
const progressLogInterval = 10 * time.Second

// SeedAction mirrors SearchIndexSeedAction.
type SeedAction struct {
	Index         string
	TargetVersion VersionRecord
	Swap          bool
	Reason        string // index_created | index_empty | schema_changed | last_run_failed
}

/* -------------------------------- planning -------------------------------- */

// CreateSeedPlan mirrors createSeedPlan: which indexes need data, and which
// version it goes into.
func CreateSeedPlan(ctx context.Context, c *Context) ([]SeedAction, error) {
	definitions := c.Indexes.Values()
	actions := []SeedAction{}
	if len(definitions) == 0 {
		return actions, nil
	}
	records, err := ListIndexRecords(ctx, c, filter("name", anyStrings(c.Indexes.Names())), utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
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
	for _, definition := range definitions {
		record, ok := byName[definition.Name]
		if !ok {
			continue
		}
		versions := versionsByIndexID[record.ID]
		var pending *VersionRecord
		for i := range versions {
			v := versions[i]
			if record.ActiveVersion == nil || v.Version > *record.ActiveVersion {
				if pending == nil || v.Version > pending.Version {
					pending = &versions[i]
				}
			}
		}
		if pending != nil {
			actions = append(actions, SeedAction{Index: definition.Name, TargetVersion: *pending, Swap: true, Reason: "schema_changed"})
			continue
		}
		var activeVersion *VersionRecord
		for i := range versions {
			if record.ActiveVersion != nil && versions[i].Version == *record.ActiveVersion {
				activeVersion = &versions[i]
				break
			}
		}
		if activeVersion == nil {
			continue
		}
		if activeVersion.Status == StateError {
			count, err := countDocuments(ctx, c, definition, activeVersion.PhysicalName)
			if err != nil {
				return nil, err
			}
			reason := "last_run_failed"
			if count == 0 {
				reason = "index_empty"
			}
			actions = append(actions, SeedAction{Index: definition.Name, TargetVersion: *activeVersion, Reason: reason})
			continue
		}
		if activeVersion.Status != StateReady {
			actions = append(actions, SeedAction{Index: definition.Name, TargetVersion: *activeVersion, Reason: "index_created"})
			continue
		}
		count, err := countDocuments(ctx, c, definition, activeVersion.PhysicalName)
		if err != nil {
			return nil, err
		}
		if count == 0 {
			actions = append(actions, SeedAction{Index: definition.Name, TargetVersion: *activeVersion, Reason: "index_empty"})
		}
	}
	return actions, nil
}

// ExecuteSeedPlan mirrors executeSeedPlan: one seed per index, each under
// the index's seed lock.
func ExecuteSeedPlan(ctx context.Context, c *Context, actions []SeedAction) error {
	for _, action := range actions {
		definition, err := RetrieveIndexDefinition(c.Indexes, action.Index)
		if err != nil {
			return err
		}
		withIndexLock(ctx, c, definition.Name, func(ctx context.Context) error {
			return runSeed(ctx, c, definition, action)
		})
	}
	return nil
}

func updateVersionStatus(ctx context.Context, c *Context, id, status string) error {
	_, err := c.VersionService.Update(ctx, filter("selector", filter("id", id), "data", filter("status", status)))
	return err
}

func runSeed(ctx context.Context, c *Context, definition *searchtypes.ResolvedDefinition, action SeedAction) error {
	target := action.TargetVersion
	sync, err := startSync(ctx, c, startSyncInput{versionID: target.ID, resume: action.Reason != "index_empty", resumable: true})
	if err != nil {
		return err
	}
	startedAt := time.Now()
	c.Logger.Info(fmt.Sprintf(`[Search] Seeding "%s" (%s) into "%s"%s`, definition.Name, action.Reason, target.PhysicalName, formatResume(sync.LastKey)))
	if err := updateVersionStatus(ctx, c, target.ID, StateBuilding); err != nil {
		return err
	}
	fail := func(err error) error {
		failSync(ctx, c, sync.ID, target.ID, err)
		return err
	}
	documentsSynced, written, err := streamSeed(ctx, c, streamSeedInput{index: definition, syncID: sync.ID, targetIndex: target.PhysicalName, lastKey: sync.LastKey, syncedBefore: sync.DocumentsSynced})
	if err != nil {
		return fail(err)
	}
	if err := catchUp(ctx, c, definition, target, sync.StartedAt, sync.JobID); err != nil {
		return fail(err)
	}
	if err := updateVersionStatus(ctx, c, target.ID, StateReady); err != nil {
		return fail(err)
	}
	var previousVersion *int
	if action.Swap {
		record, err := retrieveIndexRecord(ctx, c, definition.Name)
		if err != nil {
			return fail(err)
		}
		previousVersion = record.ActiveVersion
		c.Logger.Info(fmt.Sprintf(`[Search] Making "%s" version %d active`, definition.Name, target.Version))
		if _, err := c.IndexService.Update(ctx, filter("selector", filter("name", definition.Name), "data", filter("active_version", float64(target.Version)))); err != nil {
			return fail(err)
		}
		if c.ActiveVersionCache != nil {
			c.ActiveVersionCache.Set(definition.Name, ActiveIndexVersion{PhysicalName: target.PhysicalName, Provider: target.Provider, Version: target.Version})
		}
	}
	if err := completeSync(ctx, c, sync.ID, documentsSynced); err != nil {
		return fail(err)
	}
	if action.Swap {
		CleanupAfterSwap(ctx, c, target.SearchIndexID, target.Version, previousVersion)
	}
	c.Logger.Info(fmt.Sprintf(`[Search] Seeded "%s": %s in %s%s`, definition.Name, formatCount(written), formatElapsed(time.Since(startedAt)), formatTotal(documentsSynced, written)))
	return nil
}

// catchUp mirrors catchUp: a second pass over what changed since the first
// one started, into the same target.
func catchUp(ctx context.Context, c *Context, definition *searchtypes.ResolvedDefinition, target VersionRecord, since time.Time, jobID string) error {
	sync, err := startSync(ctx, c, startSyncInput{versionID: target.ID, jobID: jobID, resume: false, resumable: false})
	if err != nil {
		return err
	}
	documentsSynced, _, err := streamSeed(ctx, c, streamSeedInput{index: definition, syncID: sync.ID, targetIndex: target.PhysicalName, catchup: &searchtypes.Catchup{Since: since}})
	if err != nil {
		failSync(ctx, c, sync.ID, target.ID, err)
		return err
	}
	if err := completeSync(ctx, c, sync.ID, documentsSynced); err != nil {
		failSync(ctx, c, sync.ID, target.ID, err)
		return err
	}
	c.Logger.Info(fmt.Sprintf(`[Search] Caught up "%s": %s synced since %s`, definition.Name, formatCount(documentsSynced), dal.FormatJSDate(since)))
	return nil
}

// ReindexIndexes mirrors reindexIndexes.
func ReindexIndexes(ctx context.Context, c *Context, input searchtypes.ReindexInput) (searchtypes.ReindexResult, error) {
	names := input.Index
	if len(names) == 0 {
		names = c.Indexes.Names()
	}
	definitions := make([]*searchtypes.ResolvedDefinition, len(names))
	for i, name := range names {
		d, err := RetrieveIndexDefinition(c.Indexes, name)
		if err != nil {
			return searchtypes.ReindexResult{}, err
		}
		definitions[i] = d
	}
	jobID := RandomUUID()
	for _, definition := range definitions {
		run := func(ctx context.Context) error { return reindexOne(ctx, c, definition, jobID, input) }
		var err error
		if c.Locking != nil {
			err = c.Locking.Execute(ctx, "search:seed:"+definition.Name, run)
		} else {
			err = run(ctx)
		}
		if err != nil {
			return searchtypes.ReindexResult{}, err
		}
	}
	return searchtypes.ReindexResult{JobID: jobID, Indexes: names}, nil
}

func reindexOne(ctx context.Context, c *Context, definition *searchtypes.ResolvedDefinition, jobID string, input searchtypes.ReindexInput) error {
	provider, err := c.Providers.Retrieve(definition.Provider)
	if err != nil {
		return err
	}
	record, err := retrieveIndexRecord(ctx, c, definition.Name)
	if err != nil {
		return err
	}
	previousVersion := record.ActiveVersion
	if err := CleanupStaleVersions(ctx, c, record); err != nil {
		return err
	}
	scoped := input.Filters != nil || input.Since != nil
	strategy := input.Strategy
	if strategy == "" {
		strategy = "swap"
	}
	useSwap := strategy == "swap" && !scoped

	var target VersionRecord
	if useSwap {
		target, err = createPendingVersion(ctx, c, definition, record, provider)
	} else {
		target, err = retrieveActiveVersion(ctx, c, definition, record)
	}
	if err != nil {
		return err
	}
	if useSwap {
		if err := CleanupAbandonedVersions(ctx, c, record); err != nil {
			return err
		}
	}
	sync, err := startSync(ctx, c, startSyncInput{versionID: target.ID, jobID: jobID, filters: input.Filters, resume: !scoped, resumable: !scoped})
	if err != nil {
		return err
	}
	startedAt := time.Now()
	c.Logger.Info(fmt.Sprintf(`[Search] Reindexing "%s" into "%s"%s`, definition.Name, target.PhysicalName, formatResume(sync.LastKey)))
	if err := updateVersionStatus(ctx, c, target.ID, StateBuilding); err != nil {
		return err
	}
	fail := func(err error) error {
		failSync(ctx, c, sync.ID, target.ID, err)
		return err
	}
	if !useSwap && sync.LastKey == nil && !scoped {
		err := RetryOnRateLimit(ctx, fmt.Sprintf(`clearing "%s"`, definition.Name), c.Logger, func() error {
			task, err := provider.ClearIndex(ctx, target.PhysicalName)
			if err != nil {
				return err
			}
			if _, err := AssertTaskAccepted(task, definition.Name); err != nil {
				return err
			}
			return settle(ctx, c, definition, task)
		})
		if err != nil {
			return fail(err)
		}
	}
	var catchup *searchtypes.Catchup
	if input.Since != nil {
		catchup = &searchtypes.Catchup{Since: *input.Since}
	}
	documentsSynced, written, err := streamSeed(ctx, c, streamSeedInput{index: definition, syncID: sync.ID, targetIndex: target.PhysicalName, filters: input.Filters, lastKey: sync.LastKey, syncedBefore: sync.DocumentsSynced, catchup: catchup})
	if err != nil {
		return fail(err)
	}
	if !scoped {
		if err := catchUp(ctx, c, definition, target, sync.StartedAt, jobID); err != nil {
			return fail(err)
		}
	}
	if err := updateVersionStatus(ctx, c, target.ID, StateReady); err != nil {
		return fail(err)
	}
	if useSwap {
		c.Logger.Info(fmt.Sprintf(`[Search] Making "%s" version %d active`, definition.Name, target.Version))
		if _, err := c.IndexService.Update(ctx, filter("selector", filter("id", record.ID), "data", filter("active_version", float64(target.Version)))); err != nil {
			return fail(err)
		}
		if c.ActiveVersionCache != nil {
			c.ActiveVersionCache.Set(definition.Name, ActiveIndexVersion{PhysicalName: target.PhysicalName, Provider: target.Provider, Version: target.Version})
		}
	}
	if err := completeSync(ctx, c, sync.ID, documentsSynced); err != nil {
		return fail(err)
	}
	if useSwap {
		CleanupAfterSwap(ctx, c, record.ID, target.Version, previousVersion)
	}
	c.Logger.Info(fmt.Sprintf(`[Search] Reindexed "%s": %s in %s%s`, definition.Name, formatCount(written), formatElapsed(time.Since(startedAt)), formatTotal(documentsSynced, written)))
	return nil
}

func createPendingVersion(ctx context.Context, c *Context, definition *searchtypes.ResolvedDefinition, record IndexRecord, provider searchtypes.Provider) (VersionRecord, error) {
	take := 1
	versions, err := ListVersionRecords(ctx, c, filter("search_index_id", record.ID), utilsmodulessdk.FindConfig{Order: []dal.OrderBy{{Field: "version", Direction: "DESC"}}, Take: &take})
	if err != nil {
		return VersionRecord{}, err
	}
	version := 1
	if len(versions) > 0 {
		version = versions[0].Version + 1
	} else if record.ActiveVersion != nil {
		version = *record.ActiveVersion + 1
	}
	physicalName := VersionPhysicalName(definition, version)
	if _, err := provider.UpsertIndex(ctx, definition.WithPhysicalName(physicalName)); err != nil {
		return VersionRecord{}, err
	}
	created, err := c.VersionService.Create(ctx, []*dal.Record{filter(
		"search_index_id", record.ID,
		"version", float64(version),
		"provider", definition.Provider,
		"physical_name", physicalName,
		"definition_hash", definition.DefinitionHash,
		"status", StatePending,
	)})
	if err != nil {
		return VersionRecord{}, err
	}
	out := toVersionRecord(created[0])
	if out.SearchIndexID == "" {
		out.SearchIndexID = record.ID
	}
	return out, nil
}

func retrieveActiveVersion(ctx context.Context, c *Context, definition *searchtypes.ResolvedDefinition, record IndexRecord) (VersionRecord, error) {
	if record.ActiveVersion == nil {
		return VersionRecord{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, fmt.Sprintf(`Search index "%s" has no active version to reindex in place`, definition.Name))
	}
	versions, err := ListVersionRecords(ctx, c, filter("search_index_id", record.ID, "version", float64(*record.ActiveVersion)), utilsmodulessdk.FindConfig{})
	if err != nil {
		return VersionRecord{}, err
	}
	if len(versions) == 0 {
		return VersionRecord{}, fmt.Errorf("Cannot read properties of undefined (reading 'id')")
	}
	return versions[0], nil
}

/* -------------------------------- streaming ------------------------------- */

type streamSeedInput struct {
	index        *searchtypes.ResolvedDefinition
	syncID       string
	targetIndex  string
	filters      *dal.Record
	lastKey      *string
	syncedBefore int
	catchup      *searchtypes.Catchup
}

// SeedProgressLost mirrors SearchSeedProgressLost.
type SeedProgressLost struct{ *common.MedusaError }

// streamSeed mirrors streamSeed: the definition's seed streamed in batches,
// the sync row's last_key advanced per batch.
func streamSeed(ctx context.Context, c *Context, in streamSeedInput) (int, int, error) {
	provider, err := c.Providers.Retrieve(in.index.Provider)
	if err != nil {
		return 0, 0, err
	}
	batchSize := c.BatchSize
	if batchSize == 0 {
		batchSize = DefaultReindexBatchSize
	}
	var buffer []*dal.Record
	written := 0
	total := func() int { return in.syncedBefore + written }
	var lastKey string
	startedAt := time.Now()
	lastLoggedAt := startedAt
	lastLoggedCount := 0

	reportProgress := func(force bool) {
		now := time.Now()
		if !force && (written == lastLoggedCount || now.Sub(lastLoggedAt) < progressLogInterval) {
			return
		}
		elapsed := now.Sub(startedAt)
		elapsedSec := math.Max(elapsed.Seconds(), 0.001)
		rate := math.Round(float64(written) / elapsedSec)
		cursor := ""
		if lastKey != "" {
			cursor = ", last key " + lastKey
		}
		c.Logger.Info(fmt.Sprintf(`[Search] "%s": %s synced in %s (%d/s)%s`, in.index.Name, formatCount(written), formatElapsed(elapsed), int(rate), cursor))
		lastLoggedAt = now
		lastLoggedCount = written
	}

	flushUpserts := func() error {
		if len(buffer) == 0 {
			return nil
		}
		batch := buffer
		buffer = nil
		err := RetryOnRateLimit(ctx, fmt.Sprintf(`writing %d document(s) to "%s"`, len(batch), in.index.Name), c.Logger, func() error {
			task, err := provider.UpsertDocuments(ctx, in.targetIndex, in.index, batch)
			if err != nil {
				return err
			}
			if _, err := AssertTaskAccepted(task, in.index.Name); err != nil {
				return err
			}
			return settle(ctx, c, in.index, task)
		})
		if err != nil {
			return err
		}
		written += len(batch)
		last, _ := batch[len(batch)-1].Get(in.index.PrimaryKey)
		lastKey = searchutils.JSString(last)
		if _, err := c.SyncService.Update(ctx, filter("selector", filter("id", in.syncID), "data", filter("documents_synced", float64(total()), "last_key", lastKey))); err != nil {
			return err
		}
		reportProgress(lastLoggedCount == 0)
		return nil
	}

	applyDelete := func(filters *dal.Record) error {
		if err := flushUpserts(); err != nil {
			return err
		}
		err := RetryOnRateLimit(ctx, fmt.Sprintf(`deleting documents from "%s"`, in.index.Name), c.Logger, func() error {
			task, err := provider.DeleteDocuments(ctx, in.targetIndex, filters)
			if err != nil {
				return err
			}
			if _, err := AssertTaskAccepted(task, in.index.Name); err != nil {
				return err
			}
			return settle(ctx, c, in.index, task)
		})
		if err != nil {
			return SeedProgressLost{common.NewMedusaError(common.MedusaErrorTypes.UNEXPECTED_STATE, fmt.Sprintf(`Seed of "%s" failed while deleting documents, so its progress cannot be resumed: %s`, in.index.Name, err.Error()))}
		}
		return nil
	}

	seed := in.index.Seed
	if seed == nil {
		return 0, 0, fmt.Errorf("index.seed is not a function or its return value is not async iterable")
	}
	err = seed(ctx, searchtypes.SeedContext{
		IngestionContext: searchtypes.IngestionContext{Container: c.Container(), Index: in.index},
		Filters:          in.filters,
		LastKey:          in.lastKey,
		Catchup:          in.catchup,
	}, func(mutations []searchtypes.Mutation) error {
		for _, mutation := range mutations {
			if mutation.Action == "delete" {
				if err := applyDelete(mutation.Filters); err != nil {
					return err
				}
				continue
			}
			for _, document := range mutation.Documents {
				buffer = append(buffer, document)
				if len(buffer) >= batchSize {
					if err := flushUpserts(); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return 0, 0, err
	}
	if err := flushUpserts(); err != nil {
		return 0, 0, err
	}
	if _, err := c.SyncService.Update(ctx, filter("selector", filter("id", in.syncID), "data", filter("resumable", false))); err != nil {
		return 0, 0, err
	}
	return total(), written, nil
}

// settle mirrors settle: blocks until a write lands.
func settle(ctx context.Context, c *Context, definition *searchtypes.ResolvedDefinition, task searchtypes.Task) error {
	provider, err := c.Providers.Retrieve(definition.Provider)
	if err != nil {
		return err
	}
	waiter, ok := provider.(searchtypes.TaskWaiter)
	if !ok || task.Status == "succeeded" {
		return nil
	}
	settled, err := waiter.WaitForTask(ctx, task)
	if err != nil {
		return err
	}
	_, err = AssertTaskAccepted(settled, definition.Name)
	return err
}

/* ------------------------------ sync records ------------------------------ */

// SyncRecord mirrors SearchIndexSyncRecord (the fields the seed reads).
type SyncRecord struct {
	ID              string
	JobID           string
	Status          string
	LastKey         *string
	Resumable       bool
	DocumentsSynced int
	StartedAt       time.Time
}

func toSyncRecord(r *dal.Record) SyncRecord {
	out := SyncRecord{}
	out.ID, _ = get(r, "id").(string)
	out.JobID, _ = get(r, "job_id").(string)
	out.Status, _ = get(r, "status").(string)
	if s, ok := get(r, "last_key").(string); ok {
		out.LastKey = &s
	}
	out.Resumable, _ = get(r, "resumable").(bool)
	if n, ok := get(r, "documents_synced").(float64); ok {
		out.DocumentsSynced = int(n)
	}
	out.StartedAt, _ = get(r, "started_at").(time.Time)
	return out
}

type startSyncInput struct {
	versionID string
	jobID     string
	filters   *dal.Record
	resume    bool
	resumable bool
}

func startSync(ctx context.Context, c *Context, in startSyncInput) (SyncRecord, error) {
	var previous *SyncRecord
	if in.resume {
		p, err := findResumableSync(ctx, c, in.versionID)
		if err != nil {
			return SyncRecord{}, err
		}
		previous = p
	}
	jobID := in.jobID
	if jobID == "" {
		jobID = RandomUUID()
	}
	var lastKey any
	documentsSynced := 0
	if previous != nil {
		if previous.LastKey != nil {
			lastKey = *previous.LastKey
		}
		documentsSynced = previous.DocumentsSynced
	}
	var filters any
	if in.filters != nil {
		filters = in.filters
	}
	created, err := c.SyncService.Create(ctx, []*dal.Record{filter(
		"search_index_version_id", in.versionID,
		"job_id", jobID,
		"status", SyncProcessing,
		"filters", filters,
		"last_key", lastKey,
		"documents_synced", float64(documentsSynced),
		"resumable", in.resumable,
		"started_at", time.Now().Truncate(time.Millisecond),
	)})
	if err != nil {
		return SyncRecord{}, err
	}
	if previous != nil && isUnfinished(*previous) {
		if _, err := c.SyncService.Update(ctx, filter("selector", filter("id", previous.ID), "data", filter("status", SyncCanceled))); err != nil {
			return SyncRecord{}, err
		}
	}
	return toSyncRecord(created[0]), nil
}

func completeSync(ctx context.Context, c *Context, syncID string, documentsSynced int) error {
	_, err := c.SyncService.Update(ctx, filter("selector", filter("id", syncID), "data", filter(
		"status", SyncDone,
		"documents_synced", float64(documentsSynced),
		"completed_at", time.Now().Truncate(time.Millisecond),
	)))
	return err
}

func failSync(ctx context.Context, c *Context, syncID, versionID string, cause error) {
	data := filter("status", SyncFailed, "error", cause.Error(), "completed_at", time.Now().Truncate(time.Millisecond))
	var lost SeedProgressLost
	if errors.As(cause, &lost) {
		data.Set("resumable", false)
	}
	if _, err := c.SyncService.Update(ctx, filter("selector", filter("id", syncID), "data", data)); err != nil {
		c.Logger.Warn(err.Error())
		return
	}
	if err := updateVersionStatus(ctx, c, versionID, StateError); err != nil {
		c.Logger.Warn(err.Error())
	}
}

func isUnfinished(sync SyncRecord) bool {
	return sync.Status == SyncPending || sync.Status == SyncProcessing
}

func findResumableSync(ctx context.Context, c *Context, versionID string) (*SyncRecord, error) {
	take := 1
	rows, err := c.SyncService.List(ctx, filter("search_index_version_id", versionID), utilsmodulessdk.FindConfig{Order: []dal.OrderBy{{Field: "created_at", Direction: "DESC"}}, Take: &take})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	latest := toSyncRecord(rows[0])
	if !latest.Resumable {
		return nil, nil
	}
	if isUnfinished(latest) || latest.Status == SyncFailed {
		return &latest, nil
	}
	return nil, nil
}

/* --------------------------------- helpers -------------------------------- */

func formatThousands(n int) string {
	s := fmt.Sprint(n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

func formatCount(count int) string {
	suffix := "s"
	if count == 1 {
		suffix = ""
	}
	return fmt.Sprintf("%s document%s", formatThousands(count), suffix)
}

func formatElapsed(d time.Duration) string {
	seconds := int(math.Max(0, math.Round(float64(d.Milliseconds())/1000)))
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	minutes := seconds / 60
	rest := seconds % 60
	if minutes < 60 {
		if rest != 0 {
			return fmt.Sprintf("%dm %ds", minutes, rest)
		}
		return fmt.Sprintf("%dm", minutes)
	}
	hours := minutes / 60
	minutesRest := minutes % 60
	if minutesRest != 0 {
		return fmt.Sprintf("%dh %dm", hours, minutesRest)
	}
	return fmt.Sprintf("%dh", hours)
}

func formatResume(lastKey *string) string {
	if lastKey == nil || *lastKey == "" {
		return ""
	}
	return ", resuming after " + *lastKey
}

func formatTotal(documentsSynced, written int) string {
	if documentsSynced == written {
		return ""
	}
	return ", " + formatCount(documentsSynced) + " in total"
}

func countDocuments(ctx context.Context, c *Context, definition *searchtypes.ResolvedDefinition, physicalName string) (int, error) {
	provider, err := c.Providers.Retrieve(definition.Provider)
	if err != nil {
		return 0, err
	}
	indexes, err := provider.ListIndexes(ctx)
	if err != nil {
		return 0, err
	}
	for _, info := range indexes {
		if info.Name == physicalName {
			return info.DocumentCount, nil
		}
	}
	return 0, nil
}

// withIndexLock mirrors withIndexLock: a seed failure is logged, never
// returned.
func withIndexLock(ctx context.Context, c *Context, name string, job func(ctx context.Context) error) {
	var err error
	if c.Locking == nil {
		err = job(ctx)
	} else {
		err = c.Locking.Execute(ctx, "search:seed:"+name, job)
	}
	if err != nil {
		c.Logger.Error(fmt.Sprintf(`[Search] Failed to seed "%s": %s`, name, err.Error()))
	}
}

func retrieveIndexRecord(ctx context.Context, c *Context, name string) (IndexRecord, error) {
	records, err := ListIndexRecords(ctx, c, filter("name", name), utilsmodulessdk.FindConfig{})
	if err != nil {
		return IndexRecord{}, err
	}
	if len(records) == 0 {
		return IndexRecord{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf(`Search index "%s" has no record; run migrations to create it`, name))
	}
	return records[0], nil
}

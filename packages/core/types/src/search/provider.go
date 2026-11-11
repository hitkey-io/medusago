package search

import (
	"context"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Task mirrors SearchTask.
type Task struct {
	ID     string
	Index  string
	Status string // "enqueued" | "processing" | "succeeded" | "failed"
	Error  error
}

// IndexInfo mirrors SearchIndexInfo.
type IndexInfo struct {
	Name          string
	Provider      string
	DocumentCount int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ProviderQuery mirrors ProviderSearchQuery. Filters, Pagination and
// SearchOptions are plain objects (nil when undefined).
type ProviderQuery struct {
	Index                *ResolvedDefinition
	AttributesToRetrieve []string
	// Q is nil when undefined.
	Q             *string
	Filters       *dal.Record
	Pagination    *dal.Record
	SearchOptions *dal.Record
}

// Clone copies the query (the plain objects are shared).
func (q *ProviderQuery) Clone() *ProviderQuery {
	c := *q
	return &c
}

// Provider mirrors ISearchProvider. SearchMany is optional upstream; every
// Go provider implements it (AbstractSearchProviderService's default runs
// search for each query).
type Provider interface {
	Identifier() string
	Search(ctx context.Context, input *ProviderQuery) (*dal.Record, error)
	SearchMany(ctx context.Context, inputs []*ProviderQuery) ([]*dal.Record, error)
	UpsertIndex(ctx context.Context, index *ResolvedDefinition) (Task, error)
	DeleteIndex(ctx context.Context, index string) (Task, error)
	ListIndexes(ctx context.Context) ([]IndexInfo, error)
	UpsertDocuments(ctx context.Context, index string, definition *ResolvedDefinition, documents []*dal.Record) (Task, error)
	DeleteDocuments(ctx context.Context, index string, filters *dal.Record) (Task, error)
	ClearIndex(ctx context.Context, index string) (Task, error)
}

// TaskWaiter is the optional waitForTask of a provider applying writes
// asynchronously.
type TaskWaiter interface {
	WaitForTask(ctx context.Context, task Task) (Task, error)
}

// MigrationAction mirrors SearchIndexMigrationAction.
type MigrationAction struct {
	Action               string // "create" | "migrate" | "noop" | "drop"
	Index                string
	DefinitionHash       string
	PhysicalName         string
	Version              int
	ActivePhysicalName   string
	ActiveDefinitionHash string
	Provider             string
	PreviousProvider     string
	// PhysicalNames are the dropped physical indexes (drop).
	PhysicalNames []string
}

// ReindexInput mirrors SearchReindexInput.
type ReindexInput struct {
	Index    []string
	Strategy string
	Filters  *dal.Record
	Since    *time.Time
}

// ReindexResult mirrors SearchReindexResult.
type ReindexResult struct {
	JobID   string
	Indexes []string
}

// DeleteResult mirrors SearchIndexDeleteResult.
type DeleteResult struct {
	Index           string
	DeletedVersions int
}

package utils

import (
	"context"

	searchtypes "github.com/hitkey-io/medusago/packages/core/types/src/search"
)

// BuildEventRoutes mirrors buildEventRoutes: event name -> the indexes
// that declared it.
func BuildEventRoutes(indexes *Indexes) map[string][]string {
	routes := map[string][]string{}
	for _, definition := range indexes.Values() {
		for _, event := range definition.Events {
			routes[event] = append(routes[event], definition.Name)
		}
	}
	return routes
}

// IngestEvent mirrors ingestEvent: one event turned into writes, each
// waited on as far as its provider allows.
func IngestEvent(ctx context.Context, c *Context, event searchtypes.Event, routes map[string][]string) ([]searchtypes.Task, error) {
	names := routes[event.Name]
	tasks := []searchtypes.Task{}
	if len(names) == 0 {
		return tasks, nil
	}
	for _, name := range names {
		definition, err := RetrieveIndexDefinition(c.Indexes, name)
		if err != nil {
			return nil, err
		}
		mutations, err := definition.Consume(ctx, event, searchtypes.IngestionContext{Container: c.Container(), Index: definition})
		if err != nil {
			return nil, err
		}
		written, err := WithActiveIndexRetry(ctx, c, func() ([]searchtypes.Task, error) {
			active, err := ResolveActiveDefinition(ctx, c, name, true)
			if err != nil {
				return nil, err
			}
			provider, err := c.Providers.Retrieve(active.Provider)
			if err != nil {
				return nil, err
			}
			applied := []searchtypes.Task{}
			for _, mutation := range mutations {
				if mutation.Action == "upsert" && len(mutation.Documents) == 0 {
					continue
				}
				var task searchtypes.Task
				if mutation.Action == "upsert" {
					task, err = provider.UpsertDocuments(ctx, active.PhysicalName, active, mutation.Documents)
				} else {
					task, err = provider.DeleteDocuments(ctx, active.PhysicalName, mutation.Filters)
				}
				if err != nil {
					return nil, err
				}
				if task, err = AssertTaskAccepted(task, name); err != nil {
					return nil, err
				}
				settled, err := settleTask(ctx, provider, task, name)
				if err != nil {
					return nil, err
				}
				applied = append(applied, settled)
			}
			return applied, nil
		})
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, written...)
	}
	return tasks, nil
}

func settleTask(ctx context.Context, provider searchtypes.Provider, task searchtypes.Task, index string) (searchtypes.Task, error) {
	waiter, ok := provider.(searchtypes.TaskWaiter)
	if task.Status == "succeeded" || !ok {
		return task, nil
	}
	settled, err := waiter.WaitForTask(ctx, task)
	if err != nil {
		return task, err
	}
	return AssertTaskAccepted(settled, index)
}

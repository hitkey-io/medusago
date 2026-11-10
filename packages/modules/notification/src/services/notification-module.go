// Package services mirrors @medusajs/notification/src/services.
package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/notification/src/models"
)

// Provider is a notification provider (INotificationProvider.send).
type Provider interface {
	Send(notification *dal.Record) (*dal.Record, error)
}

// NotificationModuleService mirrors NotificationModuleService.
type NotificationModuleService struct {
	*utilsmodulessdk.MedusaService
	providers map[string]Provider

	cacheMu        sync.Mutex
	providersCache map[string]*dal.Record
}

// NewNotificationModuleService builds the service.
func NewNotificationModuleService(opts modulessdk.ServiceOptions, joinerConfig *joiner.ModuleJoinerConfig) (*NotificationModuleService, error) {
	s := &NotificationModuleService{MedusaService: utilsmodulessdk.NewMedusaService(opts.DB, models.All, joinerConfig), providers: map[string]Provider{}}
	for _, p := range opts.Providers {
		if inst, ok := p.Instance.(Provider); ok {
			s.providers[p.ID] = inst
		}
	}
	return s, nil
}

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

// getProviderForChannels mirrors NotificationProviderService.getProviderForChannels
// (enabled providers cached by channel on first use).
func (s *NotificationModuleService) getProviderForChannels(ctx context.Context) (map[string]*dal.Record, error) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if s.providersCache == nil {
		where := dal.NewRecord()
		where.Set("is_enabled", true)
		providers, err := s.Internal("NotificationProvider").List(ctx, where, utilsmodulessdk.FindConfig{})
		if err != nil {
			return nil, err
		}
		cache := map[string]*dal.Record{}
		for _, p := range providers {
			channels, _ := get(p, "channels").([]any)
			for _, c := range channels {
				cache[fmt.Sprint(c)] = p
			}
		}
		s.providersCache = cache
	}
	return s.providersCache, nil
}

// CreateNotifications mirrors createNotifications (array input).
func (s *NotificationModuleService) CreateNotifications(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	if len(data) == 0 {
		return []*dal.Record{}, nil
	}
	notifications := s.Internal("Notification")
	idempotencyKeys := []any{}
	for _, d := range data {
		if k := get(d, "idempotency_key"); k != nil && k != "" {
			idempotencyKeys = append(idempotencyKeys, k)
		}
	}
	filter := dal.NewRecord()
	filter.Set("idempotency_key", idempotencyKeys)
	already, err := notifications.List(ctx, filter, utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	exists := map[string]*dal.Record{}
	for _, n := range already {
		exists[fmt.Sprint(get(n, "idempotency_key"))] = n
	}
	providersByChannel, err := s.getProviderForChannels(ctx)
	if err != nil {
		return nil, err
	}
	type entry struct {
		provider *dal.Record
		data     *dal.Record
	}
	var toProcess []entry
	var toCreate []*dal.Record
	for _, d := range data {
		key := get(d, "idempotency_key")
		hasKey := key != nil && key != ""
		existing, found := exists[fmt.Sprint(key)]
		if hasKey && found && get(existing, "status") != "failure" {
			continue
		}
		provider := providersByChannel[fmt.Sprint(get(d, "channel"))]
		nd := dal.NewRecord()
		nd.Set("id", common.GenerateEntityID("", "noti"))
		for _, k := range d.Keys() {
			v, _ := d.Get(k)
			nd.Set(k, v)
		}
		if provider != nil {
			nd.Set("provider_id", get(provider, "id"))
		} else {
			nd.Delete("provider_id")
		}
		toProcess = append(toProcess, entry{provider: provider, data: nd})
		if !hasKey || !found {
			toCreate = append(toCreate, nd)
		}
	}
	created := []*dal.Record{}
	if len(toCreate) > 0 {
		created, err = notifications.Create(ctx, toCreate)
		if err != nil {
			return nil, err
		}
	}

	// promiseAll(..., { aggregateErrors: true }): every send settles, the
	// rejections are joined in input order into one plain Error
	var mu sync.Mutex
	var toUpdate []*dal.Record
	errs := make([]error, len(toProcess))
	var wg sync.WaitGroup
	for i, e := range toProcess {
		wg.Add(1)
		go func(i int, e entry) {
			defer wg.Done()
			push := func(err error) {
				mu.Lock()
				toUpdate = append(toUpdate, e.data)
				errs[i] = err
				mu.Unlock()
			}
			if e.provider == nil || get(e.provider, "is_enabled") != true {
				e.data.Set("status", "failure")
				msg := fmt.Sprintf("Could not find a notification provider for channel: %v for notification id %v", get(e.data, "channel"), get(e.data, "id"))
				if e.provider != nil {
					msg = fmt.Sprintf("Notification provider %v is not enabled. To enable it, configure it as a provider in the notification module options.", get(e.provider, "id"))
				}
				push(common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, msg))
				return
			}
			handler := s.providers[fmt.Sprint(get(e.provider, "id"))]
			if handler == nil {
				e.data.Set("status", "failure")
				push(fmt.Errorf("\nUnable to retrieve the notification provider with id: %v\nPlease make sure that the provider is registered in the container and it is configured correctly in your project configuration file.", get(e.provider, "id")))
				return
			}
			res, err := handler.Send(e.data)
			if err != nil {
				e.data.Set("status", "failure")
				push(common.NewMedusaError(common.MedusaErrorTypes.UNEXPECTED_STATE, fmt.Sprintf("Failed to send notification with id %v:\n%s", get(e.data, "id"), err.Error())))
				return
			}
			if ext, ok := res.Get("id"); ok {
				e.data.Set("external_id", ext)
			}
			e.data.Set("status", "success")
			push(nil)
		}(i, e)
	}
	wg.Wait()

	if len(toUpdate) > 0 {
		input := make([]any, len(toUpdate))
		for i, u := range toUpdate {
			input[i] = u
		}
		updatedAny, uerr := notifications.Update(ctx, input)
		if uerr != nil {
			return nil, uerr
		}
		updated := map[any]*dal.Record{}
		if list, ok := updatedAny.([]*dal.Record); ok {
			for _, u := range list {
				updated[get(u, "id")] = u
			}
		}
		for i, c := range created {
			if u := updated[get(c, "id")]; u != nil {
				created[i] = u
			}
		}
	}
	var msgs []string
	for _, e := range errs {
		if e != nil {
			msgs = append(msgs, e.Error())
		}
	}
	if len(msgs) > 0 {
		return nil, errors.New(strings.Join(msgs, "\n"))
	}
	return created, nil
}

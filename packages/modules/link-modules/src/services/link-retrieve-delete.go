package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	eventbus "github.com/hitkey-io/medusago/packages/core/utils/src/event-bus"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// keys returns primaryKey_ (the first relationship's foreign keys) and
// foreignKey_ (the second relationship's).
func (s *LinkModuleService) linkKeys() ([]string, string) {
	return strings.Split(s.Config.Relationships[0].ForeignKey, ","), s.Config.Relationships[1].ForeignKey
}

// Retrieve mirrors retrieve: the link between the primary key value(s) and
// the foreign key value.
func (s *LinkModuleService) Retrieve(ctx context.Context, primaryKeyData any, foreignKeyData string) (*dal.Record, error) {
	primary, foreign := s.linkKeys()
	if len(primary) > 1 {
		list, ok := primaryKeyData.([]any)
		if !ok || len(list) != len(primary) {
			return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Primary key data must be an array %d values", len(primary)))
		}
	}
	pk := strings.Join(primary, ",")
	f := dal.NewRecord()
	f.Set(pk, primaryKeyData)
	f.Set(foreign, foreignKeyData)
	rows, err := s.List(ctx, f, utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		pkData := fmt.Sprint(primaryKeyData)
		if list, ok := primaryKeyData.([]any); ok {
			parts := make([]string, len(list))
			for i, v := range list {
				parts[i] = fmt.Sprint(v)
			}
			pkData = strings.Join(parts, ",")
		}
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Entry %s[%s] and %s[%s] was not found", pk, pkData, foreign, foreignKeyData))
	}
	return rows[0], nil
}

// Delete mirrors delete: the links matching the entries ({ [key]: value },
// keys among the primary and foreign keys) are hard deleted and detached
// is emitted.
func (s *LinkModuleService) Delete(ctx context.Context, data []*dal.Record) error {
	primary, foreign := s.linkKeys()
	valid := map[string]bool{foreign: true}
	for _, k := range primary {
		valid[k] = true
	}
	or := make([]any, 0, len(data))
	for _, d := range data {
		for _, k := range d.Keys() {
			if !valid[k] {
				return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Invalid field name provided. Valid field names are %s", strings.Join(append(append([]string{}, primary...), foreign), ",")))
			}
		}
		or = append(or, d)
	}
	if len(or) == 0 {
		return nil
	}
	where := dal.NewRecord()
	where.Set("$or", or)
	ids, err := s.repository.Delete(ctx, where)
	if err != nil {
		return err
	}
	s.emitLinkEvent(ctx, eventbus.CommonEvents.DETACHED, ids)
	return nil
}

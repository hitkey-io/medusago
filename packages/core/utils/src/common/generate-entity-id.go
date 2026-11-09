package common

import "github.com/oklog/ulid/v2"

// GenerateEntityID mirrors generateEntityId from generate-entity-id.ts:
// returns idProperty when set, otherwise "<prefix>_<ULID>" (or a bare ULID
// when prefix is empty).
func GenerateEntityID(idProperty, prefix string) string {
	if idProperty != "" {
		return idProperty
	}
	id := ulid.Make().String()
	if prefix == "" {
		return id
	}
	return prefix + "_" + id
}

// Package translations mirrors @medusajs/utils/src/translations. The
// applyTranslations helpers read the translations through Query, so they
// live in packages/core/query/src/query (apply-translations.go).
package translations

import "github.com/hitkey-io/medusago/packages/core/utils/src/caching"

// TRANSLATION_LIST_CACHE_TAG mirrors TRANSLATION_LIST_CACHE_TAG: the cache
// tag of the cached translation lists (applyTranslations and
// GET /admin/translations).
const TRANSLATION_LIST_CACHE_TAG = "Translation:list:*"

// ListCacheOptions returns the cache option of those queries:
// { enable: true, tags: [TRANSLATION_LIST_CACHE_TAG], computeAutomaticTags: true }.
func ListCacheOptions() *caching.CacheOptions {
	o := caching.WithTags(true, TRANSLATION_LIST_CACHE_TAG)
	t := true
	o.Enable = &t
	return o
}

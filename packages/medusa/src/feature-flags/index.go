// Package featureflags mirrors @medusajs/medusa/src/feature-flags: the
// flags discovered from that directory, in file name order.
package featureflags

import ff "github.com/hitkey-io/medusago/packages/core/utils/src/feature-flags"

// BackendHmr mirrors backend-hmr.ts.
var BackendHmr = ff.FlagSettings{Key: "backend_hmr", DefaultVal: false, EnvKey: "MEDUSA_FF_BACKEND_HMR"}

// Caching mirrors caching.ts.
var Caching = ff.FlagSettings{Key: "caching", DefaultVal: false, EnvKey: "MEDUSA_FF_CACHING"}

// IndexEngine mirrors index-engine.ts.
var IndexEngine = ff.FlagSettings{Key: "index_engine", DefaultVal: false, EnvKey: "MEDUSA_FF_INDEX_ENGINE"}

// Translation mirrors translation.ts.
var Translation = ff.FlagSettings{Key: "translation", DefaultVal: false, EnvKey: "MEDUSA_FF_TRANSLATION"}

// ViewConfigurations mirrors view-configurations.ts.
var ViewConfigurations = ff.FlagSettings{Key: "view_configurations", DefaultVal: false, EnvKey: "MEDUSA_FF_VIEW_CONFIGURATIONS"}

// All lists the flags in discovery order (backend-hmr.ts, caching.ts,
// index-engine.ts, translation.ts, view-configurations.ts).
var All = []ff.FlagSettings{BackendHmr, Caching, IndexEngine, Translation, ViewConfigurations}

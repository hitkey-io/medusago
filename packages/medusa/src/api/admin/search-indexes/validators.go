package searchindexes

import (
	"regexp"

	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
)

// The ISO datetime formats of z.iso.datetime({ offset: true }) (seconds
// required, optional fraction) and with precision -1 (minutes only), as
// zod 4 builds them (core/regexes.js datetime()).
const (
	isoDateSource   = `(?:(?:\d\d[2468][048]|\d\d[13579][26]|\d\d0[48]|[02468][048]00|[13579][26]00)-02-29|\d{4}-(?:(?:0[13578]|1[02])-(?:0[1-9]|[12]\d|3[01])|(?:0[469]|11)-(?:0[1-9]|[12]\d|30)|(?:02)-(?:0[1-9]|1\d|2[0-8])))`
	isoHHMM         = `(?:[01]\d|2[0-3]):[0-5]\d`
	isoOffsetSource = `(?:Z|([+-](?:[01]\d|2[0-3]):[0-5]\d))`
)

var (
	isoDatetimeOffset        = regexp.MustCompile(`^` + isoDateSource + `T(?:` + isoHHMM + `:[0-5]\d(?:\.\d+)?` + isoOffsetSource + `)$`)
	isoDatetimeOffsetMinutes = regexp.MustCompile(`^` + isoDateSource + `T(?:` + isoHHMM + isoOffsetSource + `)$`)
)

func isoDatetime(re *regexp.Regexp) *zod.Schema {
	return zod.String().Refine(func(v any) bool {
		s, ok := v.(string)
		return ok && re.MatchString(s)
	}, "Invalid ISO datetime")
}

// AdminReindexSearchIndex mirrors AdminReindexSearchIndex.
var AdminReindexSearchIndex = zod.Object(
	// `offset: true` so a client that sends its own zone ("+02:00") is
	// accepted rather than being forced to convert to UTC first.
	zod.F("since", zod.Union(isoDatetime(isoDatetimeOffset), isoDatetime(isoDatetimeOffsetMinutes)).Optional()),
	zod.F("filters", zod.Record(zod.String(), zod.Unknown()).Optional()),
	zod.F("strategy", zod.Enum("swap", "in_place").Optional()),
).Strict()

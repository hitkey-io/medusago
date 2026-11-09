package zod

import (
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
)

// CoerceDate mirrors z.coerce.date(): the input is converted with
// `new Date(input)`; an invalid date is an invalid_type issue.
func CoerceDate() *Schema {
	s := &Schema{kind: "date"}
	s.run = func(p *payload) {
		var t time.Time
		ok := true
		switch v := p.value.(type) {
		case time.Time:
			t = v
		case string:
			t, ok = common.ParseJSDate(v)
		case float64:
			t = time.UnixMilli(int64(v)).UTC()
		case bool:
			if v {
				t = time.UnixMilli(1).UTC()
			} else {
				t = time.UnixMilli(0).UTC()
			}
		case nil:
			t = time.UnixMilli(0).UTC()
		default:
			ok = false
		}
		if !ok {
			p.issues = append(p.issues, &Issue{Code: "invalid_type", Expected: "date", Message: "Invalid input: expected date, received Date"})
			return
		}
		p.value = t
		s.withChecks(p)
	}
	return s
}

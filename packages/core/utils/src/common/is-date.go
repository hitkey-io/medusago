package common

import (
	"regexp"
	"strconv"
	"time"
)

// isoDate matches the ECMAScript date time string format (and its date-only
// forms): YYYY[-MM[-DD]][THH:mm[:ss[.sss]][Z|±HH:mm]].
var isoDate = regexp.MustCompile(`^([+-]\d{6}|\d{4})(?:-(\d{2})(?:-(\d{2}))?)?(?:[T ](\d{2}):(\d{2})(?::(\d{2})(?:\.(\d{1,9}))?)?(Z|[+-]\d{2}:?\d{2})?)?$`)

var fallbackLayouts = []string{
	time.RFC1123, time.RFC1123Z, time.RFC850, time.ANSIC, time.UnixDate, time.RubyDate,
	"Mon Jan 02 2006 15:04:05 GMT-0700", "Mon Jan 02 2006", "Jan 2, 2006", "January 2, 2006", "2006/01/02", "01/02/2006",
}

// ParseJSDate mirrors `new Date(string)` for the ECMAScript date time
// string format and the common RFC formats. Date-only forms are UTC, date
// time forms without an offset are local time.
func ParseJSDate(s string) (time.Time, bool) {
	if m := isoDate.FindStringSubmatch(s); m != nil {
		year, _ := strconv.Atoi(m[1])
		month, day := 1, 1
		if m[2] != "" {
			month, _ = strconv.Atoi(m[2])
		}
		if m[3] != "" {
			day, _ = strconv.Atoi(m[3])
		}
		if month < 1 || month > 12 || day < 1 || day > 31 {
			return time.Time{}, false
		}
		loc := time.UTC
		hour, minute, sec, nsec := 0, 0, 0, 0
		if m[4] != "" {
			hour, _ = strconv.Atoi(m[4])
			minute, _ = strconv.Atoi(m[5])
			if m[6] != "" {
				sec, _ = strconv.Atoi(m[6])
			}
			if m[7] != "" {
				frac := (m[7] + "000000000")[:9]
				nsec, _ = strconv.Atoi(frac)
				nsec = nsec / 1e6 * 1e6
			}
			if hour > 24 || minute > 59 || sec > 59 {
				return time.Time{}, false
			}
			switch tz := m[8]; {
			case tz == "":
				loc = time.Local
			case tz == "Z":
			default:
				sign := 1
				if tz[0] == '-' {
					sign = -1
				}
				digits := tz[1:]
				if len(digits) == 5 {
					digits = digits[:2] + digits[3:]
				}
				h, _ := strconv.Atoi(digits[:2])
				mm, _ := strconv.Atoi(digits[2:])
				loc = time.FixedZone("", sign*(h*3600+mm*60))
			}
		}
		t := time.Date(year, time.Month(month), day, hour, minute, sec, nsec, loc)
		if t.Day() != day {
			return time.Time{}, false
		}
		return t.UTC(), true
	}
	for _, layout := range fallbackLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

// IsDate mirrors isDate: a non-null value `new Date(value)` accepts.
func IsDate(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case time.Time:
		return true
	case string:
		_, ok := ParseJSDate(t)
		return ok
	case float64:
		return true
	}
	return false
}

// GetIsoStringFromDate mirrors GetIsoStringFromDate: new Date(v).toISOString().
func GetIsoStringFromDate(v any) string {
	switch t := v.(type) {
	case time.Time:
		return t.UTC().Format("2006-01-02T15:04:05.000Z")
	case string:
		if d, ok := ParseJSDate(t); ok {
			return d.Format("2006-01-02T15:04:05.000Z")
		}
	case float64:
		return time.UnixMilli(int64(t)).UTC().Format("2006-01-02T15:04:05.000Z")
	}
	return ""
}

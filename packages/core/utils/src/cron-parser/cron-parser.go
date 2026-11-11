// Package cronparser mirrors the parts of cron-parser (parseExpression /
// CronExpressionParser.parse and next()) used by bullmq's repeatable jobs
// and the in-memory workflow scheduler: 5 or 6 fields (optional seconds),
// lists, ranges, steps, month and weekday names, "?" and "L" (last day of
// the month); a day of the month and a day of the week that are both
// restricted match either (OR), like cron.
package cronparser

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Expression is a parsed cron expression.
type Expression struct {
	second, minute, hour, dom, month, dow []bool
	lastDom                               bool
	domStar, dowStar                      bool
	loc                                   *time.Location
	current                               time.Time
	end                                   time.Time
}

// Options mirrors the parse options used upstream.
type Options struct {
	// CurrentDate defaults to now.
	CurrentDate time.Time
	// TZ is an IANA zone name (the process zone when empty).
	TZ string
	// EndDate (zero: none) makes Next fail past it.
	EndDate time.Time
}

var monthNames = map[string]int{"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6, "jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12}
var dayNames = map[string]int{"sun": 0, "mon": 1, "tue": 2, "wed": 3, "thu": 4, "fri": 5, "sat": 6}

// Parse mirrors parseExpression(expression, options).
func Parse(expr string, opts Options) (*Expression, error) {
	fields := strings.Fields(expr)
	switch len(fields) {
	case 5:
		fields = append([]string{"0"}, fields...)
	case 6:
	default:
		return nil, fmt.Errorf("Invalid cron expression")
	}
	e := &Expression{loc: time.Local}
	if opts.TZ != "" {
		loc, err := time.LoadLocation(opts.TZ)
		if err != nil {
			return nil, err
		}
		e.loc = loc
	}
	var err error
	if e.second, err = parseField(fields[0], 0, 59, nil); err != nil {
		return nil, err
	}
	if e.minute, err = parseField(fields[1], 0, 59, nil); err != nil {
		return nil, err
	}
	if e.hour, err = parseField(fields[2], 0, 23, nil); err != nil {
		return nil, err
	}
	domField := fields[3]
	if strings.EqualFold(domField, "L") {
		e.lastDom = true
		e.dom = make([]bool, 32)
	} else if e.dom, err = parseField(domField, 1, 31, nil); err != nil {
		return nil, err
	}
	if e.month, err = parseField(fields[4], 1, 12, monthNames); err != nil {
		return nil, err
	}
	if e.dow, err = parseField(fields[5], 0, 7, dayNames); err != nil {
		return nil, err
	}
	if e.dow[7] {
		e.dow[0] = true
	}
	e.domStar = domField == "*" || domField == "?"
	e.dowStar = fields[5] == "*" || fields[5] == "?"
	e.current = opts.CurrentDate
	if e.current.IsZero() {
		e.current = time.Now()
	}
	e.end = opts.EndDate
	return e, nil
}

func parseField(field string, min, max int, names map[string]int) ([]bool, error) {
	out := make([]bool, max+1)
	for _, part := range strings.Split(field, ",") {
		step := 1
		rng := part
		if i := strings.Index(part, "/"); i >= 0 {
			s, err := strconv.Atoi(part[i+1:])
			if err != nil || s <= 0 {
				return nil, fmt.Errorf("Invalid step: %s", part)
			}
			step = s
			rng = part[:i]
		}
		lo, hi := min, max
		switch {
		case rng == "*" || rng == "?":
		case strings.Contains(rng, "-"):
			a, b, _ := strings.Cut(rng, "-")
			var err error
			if lo, err = value(a, names); err != nil {
				return nil, err
			}
			if hi, err = value(b, names); err != nil {
				return nil, err
			}
		default:
			v, err := value(rng, names)
			if err != nil {
				return nil, err
			}
			lo = v
			if !strings.Contains(part, "/") {
				hi = v
			}
		}
		if lo < min || hi > max || lo > hi {
			return nil, fmt.Errorf("Constraint error, got range %d-%d expected range %d-%d", lo, hi, min, max)
		}
		for v := lo; v <= hi; v += step {
			out[v] = true
		}
	}
	return out, nil
}

func value(s string, names map[string]int) (int, error) {
	if v, ok := names[strings.ToLower(s)]; ok {
		return v, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("Invalid characters, got value: %s", s)
	}
	return v, nil
}

func (e *Expression) dayMatches(t time.Time) bool {
	d := t.Day()
	domOK := e.dom[d]
	if e.lastDom {
		domOK = t.AddDate(0, 0, 1).Month() != t.Month()
	}
	dowOK := e.dow[int(t.Weekday())]
	switch {
	case e.domStar && e.dowStar:
		return true
	case e.domStar:
		return dowOK
	case e.dowStar:
		return domOK
	}
	return domOK || dowOK
}

// ErrOutOfRange mirrors the error thrown past the end date.
var ErrOutOfRange = errors.New("Out of the timespan range")

// Next mirrors next(): the first matching time strictly after the current
// date (second precision); it becomes the current date.
func (e *Expression) Next() (time.Time, error) {
	t := e.current.In(e.loc).Truncate(time.Second).Add(time.Second)
	limit := t.AddDate(8, 0, 0)
	for t.Before(limit) {
		if !e.month[int(t.Month())] {
			t = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, e.loc)
			continue
		}
		if !e.dayMatches(t) {
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, e.loc)
			continue
		}
		if !e.hour[t.Hour()] {
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour()+1, 0, 0, 0, e.loc)
			continue
		}
		if !e.minute[t.Minute()] {
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute()+1, 0, 0, e.loc)
			continue
		}
		if !e.second[t.Second()] {
			t = t.Add(time.Second)
			continue
		}
		if !e.end.IsZero() && t.After(e.end) {
			return time.Time{}, ErrOutOfRange
		}
		e.current = t
		return t, nil
	}
	return time.Time{}, errors.New("Invalid explicit day of month definition")
}

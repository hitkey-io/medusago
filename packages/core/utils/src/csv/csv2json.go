package csv

import (
	"math"
	"regexp"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

var dateStringRe = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d{3}Z`)

// Csv2Json mirrors json-2-csv 5.6.0 csv2json(data, { delimiter: { field } })
// with the default wrap ("), eol (\n) and JSON.parse value parsing. Like
// the original, lines are split by a character state machine (strings are
// indexed by rune, like JS code units for BMP text).
func Csv2Json(data string, field string) []*dal.Record {
	wrap, eol := `"`, "\n"
	lines := splitCsvLines([]rune(data), field, wrap, eol)
	if len(lines) == 0 {
		return []*dal.Record{}
	}
	type header struct {
		value string
		index int
	}
	var headers []header
	for i, h := range lines[0] {
		headers = append(headers, header{value: removeWrapDelimiters(h, wrap), index: i})
	}
	escaped := wrap + wrap
	out := []*dal.Record{}
	for _, line := range lines[1:] {
		values := make([]string, len(line))
		for i, v := range line {
			v = removeWrapDelimiters(v, wrap)
			values[i] = strings.ReplaceAll(v, escaped, wrap)
		}
		doc := dal.NewRecord()
		for _, h := range headers {
			var value any
			if h.index < len(values) {
				value = processRecordValue(values[h.index], wrap)
			} else {
				value = processRecordValue("undefined", wrap)
			}
			if h.value == "" {
				continue
			}
			setPathValue(doc, h.value, value)
		}
		out = append(out, doc)
	}
	return out
}

func removeWrapDelimiters(v, wrap string) string {
	r := []rune(v)
	if len(r) > 0 && string(r[0]) == wrap && string(r[len(r)-1]) == wrap {
		if len(r) <= 2 {
			return ""
		}
		return string(r[1 : len(r)-1])
	}
	return v
}

// processRecordValue mirrors processRecordValue: JSON.parse when it works
// ("12" -> 12, "true" -> true, "null" -> null), the raw string otherwise.
func processRecordValue(v string, wrap string) any {
	r := []rune(v)
	isStringRepresentation := len(r) > 0 && string(r[0]) == wrap && string(r[len(r)-1]) == wrap
	if isStringRepresentation && !dateStringRe.MatchString(v) {
		return v
	}
	parsed, err := dal.DecodeOrderedJSON([]byte(v))
	if err != nil || strings.TrimSpace(v) == "" || !validJSONText(v) {
		if v == "undefined" {
			return Undefined
		}
		return v
	}
	if f, ok := parsed.(float64); ok && math.IsInf(f, 0) {
		return v
	}
	return parsed
}

// validJSONText reports whether the whole text is a single JSON value (the
// streaming decoder accepts trailing data).
func validJSONText(v string) bool {
	return jsonValid([]byte(v))
}

func setPathValue(doc *dal.Record, kp string, value any) {
	dot, key, remaining := state(kp)
	if dot < 0 || remaining == "" {
		doc.Set(kp, value)
		return
	}
	child, _ := get(doc, key).(*dal.Record)
	if child == nil {
		child = dal.NewRecord()
		doc.Set(key, child)
	}
	setPathValue(child, remaining, value)
}

func substr(csv []rune, from, to int) string {
	if from > len(csv) {
		return ""
	}
	if to > len(csv) || to < 0 {
		to = len(csv)
	}
	if from > to {
		from, to = to, from
	}
	return string(csv[from:to])
}

func nChars(csv []rune, start, n int) string {
	if start < 0 {
		start = 0
	}
	if start >= len(csv) {
		return ""
	}
	end := start + n
	if end > len(csv) {
		end = len(csv)
	}
	return string(csv[start:end])
}

func splitCsvLines(csv []rune, field, wrap, eol string) [][]string {
	var lines [][]string
	last := len(csv) - 1
	eolLen := len([]rune(eol))
	inside, parsing, justDouble, start := false, true, false, 0
	var split []string
	at := func(i int) string {
		if i < 0 || i >= len(csv) {
			return ""
		}
		return string(csv[i])
	}
	index := 0
	for index < len(csv) {
		ch := at(index)
		before := ""
		if index > 0 {
			before = at(index - 1)
		}
		after := ""
		if index < last {
			after = at(index + 1)
		}
		next := nChars(csv, index, eolLen)
		switch {
		case (next == eol && !inside || index == last) && before == field:
			if next == eol && start == index {
				split = append(split, "")
			} else if ch == field {
				split = append(split, "")
			} else {
				split = append(split, substr(csv, start, len(csv)))
			}
			split = append(split, "")
			lines = append(lines, split)
			split = nil
			start = index + eolLen
			parsing = true
			inside = after == wrap
		case index == last && ch == field:
			split = append(split, substr(csv, start, index))
			split = append(split, "")
			lines = append(lines, split)
		case index == last || next == eol && (!inside || inside && before == wrap && !justDouble):
			to := index
			if !(index != last || before == wrap) {
				to = len(csv)
			}
			split = append(split, substr(csv, start, to))
			lines = append(lines, split)
			split = nil
			start = index + eolLen
			parsing = true
			inside = after == wrap
		case ch == wrap && before == field && !inside && !parsing:
			start = index
			inside = true
			parsing = true
			if nChars(csv, index+1, eolLen) == eol {
				index += eolLen + 1
			}
		case before == field && ch == wrap && after == eol:
			split = append(split, substr(csv, start, index-1))
			start = index
			parsing = true
			inside = true
			justDouble = true
			index++
		case (before != wrap || justDouble && before == wrap) && ch == wrap && nChars(csv, index+1, eolLen) == eol:
			inside = false
			parsing = false
		case ch == wrap && (index == 0 || nChars(csv, index-eolLen, eolLen) == eol && !inside):
			inside = true
			parsing = true
			start = index
		case ch == wrap && after == field && inside:
			split = append(split, substr(csv, start, index+1))
			start = index + 2
			inside = false
			parsing = false
		case ch == wrap && before == field && !inside && parsing:
			split = append(split, substr(csv, start, index-1))
			inside = true
			parsing = true
			start = index
		case ch == wrap && after == wrap && index != start:
			index += 2
			justDouble = true
			continue
		case ch == field && before != wrap && after != wrap && !inside && parsing:
			split = append(split, substr(csv, start, index))
			start = index + 1
		case ch == field && before == wrap && after != wrap && !parsing:
			inside = false
			parsing = true
			start = index + 1
		}
		index++
		justDouble = false
	}
	return lines
}

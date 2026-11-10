package csv

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// CsvError mirrors csv-parse's CsvError.
type CsvError struct {
	Code    string
	Message string
}

func (e *CsvError) Error() string { return e.Message }

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// ParseColumns mirrors csv-parse 7 parse(input, { columns: true,
// skip_empty_lines: true }): the first record names the columns, the other
// records become column -> value records (strings). Delimiter ",", quote
// and escape `"`, record delimiter auto-detected (\r\n, \n or \r).
func ParseColumns(input string) ([]*dal.Record, error) {
	recordDelimiter := ""
	var columns []string
	var out []*dal.Record
	var record []string
	var field strings.Builder
	quoting, wasQuoting := false, false
	line := 1
	pendingLine := false

	onRecord := func() error {
		if columns == nil {
			columns = append([]string{}, record...)
			return nil
		}
		if len(record) != len(columns) {
			return &CsvError{Code: "CSV_RECORD_INCONSISTENT_COLUMNS", Message: fmt.Sprintf("Invalid Record Length: columns length is %d, got %d on line %d", len(columns), len(record), line)}
		}
		r := dal.NewRecord()
		for i, c := range columns {
			r.Set(c, record[i])
		}
		out = append(out, r)
		return nil
	}

	for pos := 0; pos < len(input); pos++ {
		if pendingLine {
			line++
			pendingLine = false
		}
		c := input[pos]
		if quoting {
			if c == '"' {
				if pos+1 < len(input) && input[pos+1] == '"' {
					field.WriteByte('"')
					pos++
					continue
				}
				quoting = false
				wasQuoting = true
				// the next character must be a delimiter, a record delimiter or the end
				if pos+1 < len(input) {
					n := input[pos+1]
					isRecordDelim := n == '\n' || n == '\r'
					if n != ',' && !isRecordDelim {
						return nil, &CsvError{Code: "CSV_INVALID_CLOSING_QUOTE", Message: fmt.Sprintf(`Invalid Closing Quote: got "%c" at line %d instead of delimiter, record delimiter, trimable character (if activated) or comment`, n, line)}
					}
				}
				continue
			}
			// newlines inside quotes do not count as lines
			field.WriteByte(c)
			continue
		}
		if c == '"' {
			if field.Len() == 0 && !wasQuoting {
				quoting = true
				continue
			}
			column := any(len(record))
			if columns != nil && len(record) < len(columns) {
				column = columns[len(record)]
			}
			colJSON, _ := json.Marshal(column)
			return nil, &CsvError{Code: "INVALID_OPENING_QUOTE", Message: fmt.Sprintf("Invalid Opening Quote: a quote is found on field %s at line %d, value is %s", colJSON, line, jsonString(field.String()))}
		}
		// record delimiter
		delimLen := 0
		if recordDelimiter == "" {
			if c == '\r' && pos+1 < len(input) && input[pos+1] == '\n' {
				recordDelimiter = "\r\n"
			} else if c == '\n' {
				recordDelimiter = "\n"
			} else if c == '\r' {
				recordDelimiter = "\r"
			}
		}
		if recordDelimiter != "" && strings.HasPrefix(input[pos:], recordDelimiter) {
			delimLen = len(recordDelimiter)
		}
		if delimLen > 0 {
			if !(wasQuoting == false && len(record) == 0 && field.Len() == 0) {
				record = append(record, field.String())
				field.Reset()
				if err := onRecord(); err != nil {
					return nil, err
				}
			}
			record = nil
			wasQuoting = false
			pos += delimLen - 1
			pendingLine = true
			continue
		}
		if c == ',' {
			record = append(record, field.String())
			field.Reset()
			wasQuoting = false
			continue
		}
		field.WriteByte(c)
	}
	if pendingLine {
		line++
	}
	if quoting {
		return nil, &CsvError{Code: "CSV_QUOTE_NOT_CLOSED", Message: fmt.Sprintf("Quote Not Closed: the parsing is finished with an opening quote at line %d", line)}
	}
	if wasQuoting || len(record) > 0 || field.Len() > 0 {
		record = append(record, field.String())
		if err := onRecord(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

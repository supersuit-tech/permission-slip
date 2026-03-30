// Package sqldb provides shared helpers for SQL-based connectors (MySQL,
// PostgreSQL, etc). These are database-agnostic utilities that handle row
// scanning, truncation detection, and sorted key extraction.
package sqldb

import (
	"database/sql"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// ScanRows reads all rows from a *sql.Rows into a slice of column-keyed maps,
// returning the column names alongside the row data. []byte values are
// converted to strings for JSON-friendly output. The caller is responsible
// for closing rows.
//
// maxRows limits how many rows are read. When > 0, at most maxRows+1 rows
// are scanned (the extra row enables truncation detection via DetectTruncation).
// Pass 0 to read all rows without a cap.
//
// errMapper is called on the final rows.Err() to convert driver-specific
// errors to typed connector errors. Pass nil to use a generic wrapper.
func ScanRows(rows *sql.Rows, maxRows int, errMapper func(error) error) ([]string, []map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, &connectors.ExternalError{Message: fmt.Sprintf("reading column names: %v", err)}
	}

	var results []map[string]interface{}
	for rows.Next() {
		if maxRows > 0 && len(results) >= maxRows+1 {
			break
		}
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, nil, &connectors.ExternalError{Message: fmt.Sprintf("scanning row: %v", err)}
		}
		row := make(map[string]interface{}, len(columns))
		for i, col := range columns {
			if b, ok := values[i].([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = values[i]
			}
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		if errMapper != nil {
			return nil, nil, errMapper(err)
		}
		return nil, nil, &connectors.ExternalError{Message: fmt.Sprintf("iterating rows: %v", err)}
	}
	if results == nil {
		results = []map[string]interface{}{}
	}
	return columns, results, nil
}

// DetectTruncation checks whether the result set has more rows than the limit.
// If so, it trims the slice to the limit and returns truncated=true.
func DetectTruncation(results []map[string]interface{}, limit int) ([]map[string]interface{}, bool) {
	if len(results) > limit {
		return results[:limit], true
	}
	return results, false
}

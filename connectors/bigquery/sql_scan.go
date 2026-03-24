package bigquery

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// translatePlaceholders maps each ? outside strings/comments to @p0, @p1, ...
func translatePlaceholders(sql string, paramCount int) (string, error) {
	n := countParamPlaceholders(sql)
	if n == 0 {
		if paramCount > 0 {
			return "", &connectors.ValidationError{Message: "sql must use ? placeholders when params is non-empty"}
		}
		return sql, nil
	}
	if n != paramCount {
		return "", &connectors.ValidationError{Message: fmt.Sprintf("sql has %d ? placeholders but params has %d values", n, paramCount)}
	}
	var b strings.Builder
	b.Grow(len(sql) + paramCount*4)
	i := 0
	idx := 0
	for i < len(sql) {
		c := sql[i]
		switch {
		case c == '-' && i+1 < len(sql) && sql[i+1] == '-':
			b.WriteString(sql[i : i+2])
			i += 2
			for i < len(sql) && sql[i] != '\n' {
				b.WriteByte(sql[i])
				i++
			}
		case c == '/' && i+1 < len(sql) && sql[i+1] == '*':
			end := strings.Index(sql[i+2:], "*/")
			if end < 0 {
				b.WriteString(sql[i:])
				return b.String(), nil
			}
			end += i + 2
			b.WriteString(sql[i : end+2])
			i = end + 2
		case c == '\'':
			start := i
			i++
			for i < len(sql) {
				if sql[i] == '\'' {
					if i+1 < len(sql) && sql[i+1] == '\'' {
						i += 2
						continue
					}
					i++
					break
				}
				if sql[i] == '\\' && i+1 < len(sql) {
					i += 2
					continue
				}
				_, w := utf8.DecodeRuneInString(sql[i:])
				i += w
			}
			b.WriteString(sql[start:i])
		case c == '"':
			start := i
			i++
			for i < len(sql) {
				if sql[i] == '"' {
					if i+1 < len(sql) && sql[i+1] == '"' {
						i += 2
						continue
					}
					i++
					break
				}
				if sql[i] == '\\' && i+1 < len(sql) {
					i += 2
					continue
				}
				_, w := utf8.DecodeRuneInString(sql[i:])
				i += w
			}
			b.WriteString(sql[start:i])
		case c == '`':
			start := i
			i++
			for i < len(sql) && sql[i] != '`' {
				i++
			}
			if i < len(sql) {
				i++
			}
			b.WriteString(sql[start:i])
		case c == '?':
			fmt.Fprintf(&b, "@p%d", idx)
			idx++
			i++
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String(), nil
}

func countParamPlaceholders(sql string) int {
	n := 0
	i := 0
	for i < len(sql) {
		c := sql[i]
		switch {
		case c == '-' && i+1 < len(sql) && sql[i+1] == '-':
			i += 2
			for i < len(sql) && sql[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(sql) && sql[i+1] == '*':
			i += 2
			for i+1 < len(sql) {
				if sql[i] == '*' && sql[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
		case c == '\'':
			i++
			for i < len(sql) {
				if sql[i] == '\'' {
					if i+1 < len(sql) && sql[i+1] == '\'' {
						i += 2
						continue
					}
					i++
					break
				}
				if sql[i] == '\\' && i+1 < len(sql) {
					i += 2
					continue
				}
				_, w := utf8.DecodeRuneInString(sql[i:])
				i += w
			}
		case c == '"':
			i++
			for i < len(sql) {
				if sql[i] == '"' {
					if i+1 < len(sql) && sql[i+1] == '"' {
						i += 2
						continue
					}
					i++
					break
				}
				if sql[i] == '\\' && i+1 < len(sql) {
					i += 2
					continue
				}
				_, w := utf8.DecodeRuneInString(sql[i:])
				i += w
			}
		case c == '`':
			i++
			for i < len(sql) && sql[i] != '`' {
				i++
			}
			if i < len(sql) {
				i++
			}
		case c == '?':
			n++
			i++
		default:
			i++
		}
	}
	return n
}

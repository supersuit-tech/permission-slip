package bigquery

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// findBarePlaceholders returns byte offsets of all ? characters outside
// string literals (including triple-quoted), quoted identifiers, and comments.
func findBarePlaceholders(sql string) []int {
	var positions []int
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
			for i < len(sql) {
				if i+1 < len(sql) && sql[i] == '*' && sql[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
		case (c == '\'' || c == '"') && i+2 < len(sql) && sql[i+1] == c && sql[i+2] == c:
			// Triple-quoted string ("""...""" or '''...''')
			q := c
			i += 3
			for i < len(sql) {
				if i+2 < len(sql) && sql[i] == q && sql[i+1] == q && sql[i+2] == q {
					i += 3
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
			positions = append(positions, i)
			i++
		default:
			i++
		}
	}
	return positions
}

// translatePlaceholders maps each ? outside strings/comments to @p0, @p1, ...
func translatePlaceholders(sql string, paramCount int) (string, error) {
	positions := findBarePlaceholders(sql)
	n := len(positions)
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
	prev := 0
	for idx, pos := range positions {
		b.WriteString(sql[prev:pos])
		fmt.Fprintf(&b, "@p%d", idx)
		prev = pos + 1
	}
	b.WriteString(sql[prev:])
	return b.String(), nil
}

func countParamPlaceholders(sql string) int {
	return len(findBarePlaceholders(sql))
}

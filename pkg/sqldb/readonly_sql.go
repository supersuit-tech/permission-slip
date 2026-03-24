package sqldb

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// ValidateReadOnlyWarehouseSQL checks that sql is a single read-only statement:
// SELECT or WITH … SELECT, no WITH … DML, no top-level dangerous keywords outside
// parentheses, and no top-level semicolons. String literals, quoted identifiers,
// and comments are masked before keyword checks.
func ValidateReadOnlyWarehouseSQL(sql string) error {
	scrubbed := scrubSQLMasking(sql)
	s := strings.TrimSpace(scrubbed)
	if s == "" {
		return &connectors.ValidationError{Message: "missing required parameter: sql"}
	}
	u := strings.ToUpper(s)
	if err := rejectMultiStatement(u); err != nil {
		return err
	}
	if strings.HasPrefix(u, "WITH") && (len(u) == 4 || !isIdentCont(u[4])) {
		rest := strings.TrimSpace(u[len("WITH"):])
		kw, err := mainKeywordAfterCTEs(rest)
		if err != nil {
			return err
		}
		switch kw {
		case "SELECT":
			return rejectDangerousKeywords(u)
		case "INSERT", "UPDATE", "DELETE", "MERGE":
			return &connectors.ValidationError{Message: fmt.Sprintf("only SELECT queries are allowed (found %s after WITH clause)", kw)}
		default:
			return &connectors.ValidationError{Message: fmt.Sprintf("only SELECT is allowed after WITH clause (found %s)", kw)}
		}
	}
	if strings.HasPrefix(u, "SELECT") && (len(u) == 6 || !isIdentCont(u[6])) {
		return rejectDangerousKeywords(u)
	}
	return &connectors.ValidationError{Message: "only SELECT queries are allowed (query must start with SELECT or WITH … SELECT)"}
}

func scrubSQLMasking(sql string) string {
	var b strings.Builder
	b.Grow(len(sql))
	i := 0
	for i < len(sql) {
		c := sql[i]
		switch {
		case c == '-' && i+1 < len(sql) && sql[i+1] == '-':
			b.WriteByte(' ')
			b.WriteByte(' ')
			i += 2
			for i < len(sql) && sql[i] != '\n' {
				b.WriteByte(' ')
				i++
			}
		case c == '/' && i+1 < len(sql) && sql[i+1] == '*':
			b.WriteByte(' ')
			b.WriteByte(' ')
			i += 2
			for i+1 < len(sql) {
				if sql[i] == '*' && sql[i+1] == '/' {
					b.WriteByte(' ')
					b.WriteByte(' ')
					i += 2
					break
				}
				b.WriteByte(' ')
				i++
			}
		case c == '\'':
			b.WriteByte(' ')
			i++
			for i < len(sql) {
				if sql[i] == '\'' {
					if i+1 < len(sql) && sql[i+1] == '\'' {
						b.WriteByte(' ')
						b.WriteByte(' ')
						i += 2
						continue
					}
					b.WriteByte(' ')
					i++
					break
				}
				if sql[i] == '\\' && i+1 < len(sql) {
					b.WriteByte(' ')
					b.WriteByte(' ')
					i += 2
					continue
				}
				_, w := utf8.DecodeRuneInString(sql[i:])
				for k := 0; k < w; k++ {
					b.WriteByte(' ')
				}
				i += w
			}
		case c == '"':
			b.WriteByte(' ')
			i++
			for i < len(sql) {
				if sql[i] == '"' {
					if i+1 < len(sql) && sql[i+1] == '"' {
						b.WriteByte(' ')
						b.WriteByte(' ')
						i += 2
						continue
					}
					b.WriteByte(' ')
					i++
					break
				}
				if sql[i] == '\\' && i+1 < len(sql) {
					b.WriteByte(' ')
					b.WriteByte(' ')
					i += 2
					continue
				}
				_, w := utf8.DecodeRuneInString(sql[i:])
				for k := 0; k < w; k++ {
					b.WriteByte(' ')
				}
				i += w
			}
		case c == '`':
			b.WriteByte(' ')
			i++
			for i < len(sql) && sql[i] != '`' {
				b.WriteByte(' ')
				i++
			}
			if i < len(sql) {
				b.WriteByte(' ')
				i++
			}
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String()
}

func rejectMultiStatement(u string) error {
	depth := 0
	for i := 0; i < len(u); i++ {
		switch u[i] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ';':
			if depth == 0 {
				return &connectors.ValidationError{Message: "query must not contain semicolons (multi-statement not allowed)"}
			}
		}
	}
	return nil
}

func mainKeywordAfterCTEs(rest string) (string, error) {
	s := rest
	for {
		s = strings.TrimLeftFunc(s, unicode.IsSpace)
		if s == "" {
			return "", &connectors.ValidationError{Message: "incomplete WITH clause"}
		}
		if !isIdentStart(s[0]) {
			return "", &connectors.ValidationError{Message: "invalid WITH clause syntax"}
		}
		j := 1
		for j < len(s) && isIdentCont(s[j]) {
			j++
		}
		s = strings.TrimLeftFunc(s[j:], unicode.IsSpace)
		if !strings.HasPrefix(s, "AS") {
			return "", &connectors.ValidationError{Message: "invalid WITH clause: expected AS"}
		}
		if len(s) > 2 && isIdentCont(s[2]) {
			return "", &connectors.ValidationError{Message: "invalid WITH clause: expected AS keyword"}
		}
		s = strings.TrimLeftFunc(s[2:], unicode.IsSpace)
		if len(s) == 0 || s[0] != '(' {
			return "", &connectors.ValidationError{Message: "invalid WITH clause: expected ("}
		}
		end, err := matchingParen(s, 0)
		if err != nil {
			return "", err
		}
		s = strings.TrimLeftFunc(s[end+1:], unicode.IsSpace)
		if strings.HasPrefix(s, ",") {
			s = strings.TrimLeftFunc(s[1:], unicode.IsSpace)
			continue
		}
		break
	}
	s = strings.TrimLeftFunc(s, unicode.IsSpace)
	if s == "" {
		return "", &connectors.ValidationError{Message: "missing query after WITH clause"}
	}
	if !isIdentStart(s[0]) {
		return "", &connectors.ValidationError{Message: "invalid token after WITH clause"}
	}
	j := 1
	for j < len(s) && isIdentCont(s[j]) {
		j++
	}
	return s[:j], nil
}

func matchingParen(u string, openIdx int) (int, error) {
	if openIdx >= len(u) || u[openIdx] != '(' {
		return -1, &connectors.ValidationError{Message: "expected opening parenthesis"}
	}
	depth := 0
	for i := openIdx; i < len(u); i++ {
		switch u[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return -1, &connectors.ValidationError{Message: "unbalanced parentheses in WITH clause"}
}

func isIdentStart(b byte) bool {
	return (b >= 'A' && b <= 'Z') || b == '_'
}

func isIdentCont(b byte) bool {
	return isIdentStart(b) || (b >= '0' && b <= '9')
}

func rejectDangerousKeywords(u string) error {
	depth := 0
	for i := 0; i < len(u); {
		switch u[i] {
		case '(':
			depth++
			i++
			continue
		case ')':
			if depth > 0 {
				depth--
			}
			i++
			continue
		}
		if depth > 0 {
			i++
			continue
		}
		if !isIdentStart(u[i]) {
			i++
			continue
		}
		j := i + 1
		for j < len(u) && isIdentCont(u[j]) {
			j++
		}
		word := u[i:j]
		if isForbiddenTopLevelKeyword(word) {
			return &connectors.ValidationError{Message: fmt.Sprintf("query contains disallowed statement or keyword: %s", word)}
		}
		i = j
	}
	return nil
}

func isForbiddenTopLevelKeyword(word string) bool {
	switch word {
	case "INSERT", "UPDATE", "DELETE", "MERGE", "CREATE", "DROP", "ALTER", "TRUNCATE", "GRANT", "REVOKE":
		return true
	default:
		return false
	}
}

// SPDX-License-Identifier: Apache-2.0

package mcpserver

import "strings"

// luceneOperatorChars are characters whose presence signals the caller is using
// Lucene query syntax intentionally; we leave such queries untouched.
const luceneOperatorChars = `"+-:*?`

// normalizeCodeQuery wraps a multi-word query in double quotes for exact-phrase
// matching, unless the caller opted out via tokenized or the query already
// carries Lucene operator syntax. It returns the (possibly trimmed) query and
// reports autoQuoted=true only when it actually wrapped the query, so callers
// can surface a visible note.
func normalizeCodeQuery(query string, tokenized bool) (string, bool) {
	trimmed := strings.TrimSpace(query)
	if tokenized || trimmed == "" {
		return trimmed, false
	}
	if !isMultiWord(trimmed) {
		return trimmed, false
	}
	if strings.ContainsAny(trimmed, luceneOperatorChars) {
		return trimmed, false
	}
	if hasBooleanOperator(trimmed) {
		return trimmed, false
	}
	return `"` + trimmed + `"`, true
}

// hasBooleanOperator reports whether the query contains a standalone uppercase
// Lucene boolean token (AND, OR, NOT). It matches whitespace-delimited tokens
// only, so ordinary words like "Android" do not count.
func hasBooleanOperator(query string) bool {
	for _, field := range strings.Fields(query) {
		switch field {
		case "AND", "OR", "NOT":
			return true
		}
	}
	return false
}

// isMultiWord reports whether the query contains more than one whitespace-
// delimited token.
func isMultiWord(query string) bool {
	return len(strings.Fields(query)) > 1
}

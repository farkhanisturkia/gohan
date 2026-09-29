package utils

import (
	"strings"
	"unicode"
)

func splitWords(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
}

func splitCamel(w string) string {
	runes := []rune(w)
	var b strings.Builder

	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if !unicode.IsUpper(prev) || nextLower {
				b.WriteByte('-')
			}
		}
		b.WriteRune(r)
	}

	return b.String()
}

func camelWords(s string) []string {
	words := splitWords(s)
	var parts []string

	for _, w := range words {
		for _, p := range strings.Split(splitCamel(w), "-") {
			if p != "" {
				parts = append(parts, p)
			}
		}
	}

	return parts
}

func ToKebabCase(s string) string {
	parts := camelWords(s)
	for i, p := range parts {
		parts[i] = strings.ToLower(p)
	}
	return strings.Join(parts, "-")
}

func ToSnakeCase(s string) string {
	parts := camelWords(s)
	for i, p := range parts {
		parts[i] = strings.ToLower(p)
	}
	return strings.Join(parts, "_")
}

func ToCamelCase(s string) string {
	parts := camelWords(s)
	var b strings.Builder

	for i, p := range parts {
		if i == 0 {
			b.WriteString(strings.ToLower(p[:1]) + p[1:])
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]) + p[1:])
	}

	return b.String()
}

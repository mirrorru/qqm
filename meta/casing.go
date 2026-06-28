// Created at 2026-06-28
package meta

import (
	"strings"
	"unicode"
)

// SplitCamelCase разбивает строку по смене регистра.
func SplitCamelCase(s string) []string {
	if s == "" {
		return nil
	}

	var result []string
	runes := []rune(s)
	start := 0

	for i := 1; i < len(runes); i++ {
		if unicode.IsUpper(runes[i]) {
			if unicode.IsLower(runes[i-1]) {
				result = append(result, string(runes[start:i]))
				start = i
			} else if i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
				result = append(result, string(runes[start:i]))
				start = i
			}
		}
	}

	result = append(result, string(runes[start:]))
	return result
}

// ToSnakeCase преобразует CamelCase в snake_case.
func ToSnakeCase(s string) string {
	split := SplitCamelCase(s)
	for idx := range split {
		split[idx] = strings.ToLower(split[idx])
	}
	return strings.Join(split, "_")
}

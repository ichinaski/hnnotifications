package main

import (
	"strings"
	"unicode"
)

func Keywords(s string) []string {
	if s == "" {
		return []string{}
	}

	s = strings.ToLower(s) // case insensitive matches

	// Split by non letter/number characters
	f := func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	}
	return strings.FieldsFunc(s, f)
}

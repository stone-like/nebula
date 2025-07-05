package tools

import (
	"strings"
)

// CleanControlCharacters は文字列から制御文字を除去する
func CleanControlCharacters(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 32 && r != '\t' && r != '\n' && r != '\r' {
			return -1 // 制御文字を除去
		}
		return r
	}, s)
}

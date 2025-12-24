package app

import (
	"regexp"
)

var (
	hanToLatin          = regexp.MustCompile(`([\p{Han}])([A-Za-z0-9])`)
	latinToHan          = regexp.MustCompile(`([A-Za-z0-9])([\p{Han}])`)
	hanToLatinMidPunct  = regexp.MustCompile(`([\p{Han}])([-/]+)([A-Za-z0-9])`)
	latinToHanMidPunct  = regexp.MustCompile(`([A-Za-z0-9])([-/]+)([\p{Han}])`)
	hanToLatinOpenPunct = regexp.MustCompile(`([\p{Han}])([\(\[\{'""]+)([A-Za-z0-9])`)
	latinToHanOpenPunct = regexp.MustCompile(`([A-Za-z0-9])([\(\[\{'""]+)([\p{Han}])`)
	hanToLatinPunct     = regexp.MustCompile(`([\p{Han}])([,.;:!?\)\]\}]+)([A-Za-z0-9])`)
	latinToHanPunct     = regexp.MustCompile(`([A-Za-z0-9])([,.;:!?\)\]\}]+)([\p{Han}])`)
)

func formatThreadMessage(content string) string {
	if content == "" {
		return content
	}

	content = hanToLatinMidPunct.ReplaceAllString(content, "$1 $2 $3")
	content = latinToHanMidPunct.ReplaceAllString(content, "$1 $2 $3")
	content = hanToLatinOpenPunct.ReplaceAllString(content, "$1 $2$3")
	content = latinToHanOpenPunct.ReplaceAllString(content, "$1 $2$3")
	content = hanToLatinPunct.ReplaceAllString(content, "$1$2 $3")
	content = latinToHanPunct.ReplaceAllString(content, "$1$2 $3")
	content = hanToLatin.ReplaceAllString(content, "$1 $2")
	content = latinToHan.ReplaceAllString(content, "$1 $2")

	return content
}

func formatThreadTitle(title string) string {
	return formatThreadMessage(title)
}

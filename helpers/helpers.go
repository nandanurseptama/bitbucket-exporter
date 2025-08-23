package helpers

import "strings"

func BoolToString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

// replace all string with replacer
//
//	// example
//	// output will be `test/129/`
//	newStr := helpers.StrReplace("test/:a/", map[string]{":a" : "129"})
func StrReplace(value string, replacer map[string]string) string {
	newStr := value
	for old, new := range replacer {
		newStr = strings.ReplaceAll(newStr, old, new)
	}

	return newStr
}

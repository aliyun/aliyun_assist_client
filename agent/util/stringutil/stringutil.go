package stringutil

import (
	"errors"
	"strings"
)

// ReplaceMarkedFields finds substrings delimited by the start and end markers,
// removes the markers, and replaces the text between the markers with the result
// of calling the fieldReplacer function on that text substring. For example, if
// the input string is:  "a string with <a>text</a> marked"
// the startMarker is:   "<a>"
// the end marker is:    "</a>"
// and fieldReplacer is: strings.ToUpper
// then the output will be: "a string with TEXT marked"
func ReplaceMarkedFields(str, startMarker, endMarker string, fieldReplacer func(string) string) (newStr string, err error) {
	startIndex := strings.Index(str, startMarker)
	newStr = ""
	for startIndex >= 0 {
		newStr += str[:startIndex]
		fieldStart := str[startIndex+len(startMarker):]
		endIndex := strings.Index(fieldStart, endMarker)
		if endIndex < 0 {
			err = errors.New("Found startMarker without endMarker!")
			return
		}
		field := fieldStart[:endIndex]
		transformedField := fieldReplacer(field)
		newStr += transformedField
		str = fieldStart[endIndex+len(endMarker):]
		startIndex = strings.Index(str, startMarker)
	}
	newStr += str
	return newStr, nil
}

func CleanupNewLines(s string) string {
	return strings.Replace(strings.Replace(s, "\n", "", -1), "\r", "", -1)
}

// CleanupJSONField converts a text to a json friendly text as follows:
// - converts multi-line fields to single line by removing all but the first line
// - escapes special characters
// - truncates remaining line to length no more than maxSummaryLength
func CleanupJSONField(field string) string {
	res := field
	endOfLinePos := strings.Index(res, "\n")
	if endOfLinePos >= 0 {
		res = res[0:endOfLinePos]
	}
	res = strings.Replace(res, `\`, `\\`, -1)
	res = strings.Replace(res, `"`, `\"`, -1)
	res = strings.Replace(res, "\t", `\t`, -1)
	return res
}

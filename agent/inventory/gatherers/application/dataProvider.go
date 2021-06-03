package application

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
)

const (
	maxSummaryLength              = 100
	ApplicationCountLimit         = 1000
	ApplicationCountLimitExceeded = "Application Count Limit Exceeded"
)

func collectApplicationData(config model.Config) (appData []model.ApplicationData) {
	return collectPlatformDependentApplicationData()
}

// convertEntriesToJsonArray converts a series of comma separated json objects
// to an array of objects. For example, if entries is:
// {"k1":"v1","k2":"v2"},{"s1":"t1"},
// then this method will return
// [{"k1":"v1","k2":"v2"},{"s1":"t1"}]
func convertEntriesToJsonArray(entries string) string {
	//trim spaces
	str := strings.TrimSpace(entries)

	//remove last ',' from string
	str = strings.TrimSuffix(str, ",")

	//add "[" in beginning & "]" at the end to create valid json string
	str = fmt.Sprintf("[%v]", str)

	return str
}

// replaceMarkedFields finds substrings delimited by the start and end markers,
// removes the markers, and replaces the text between the markers with the result
// of calling the fieldReplacer function on that text substring. For example, if
// the input string is:  "a string with <a>text</a> marked"
// the startMarker is:   "<a>"
// the end marker is:    "</a>"
// and fieldReplacer is: strings.ToUpper
// then the output will be: "a string with TEXT marked"
func replaceMarkedFields(str, startMarker, endMarker string, fieldReplacer func(string) string) (newStr string, err error) {
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

// cleanupJSONField converts a text to a json friendly text as follows:
// - converts multi-line fields to single line by removing all but the first line
// - escapes special characters
// - truncates remaining line to length no more than maxSummaryLength
func cleanupJSONField(field string) string {
	res := field
	endOfLinePos := strings.Index(res, "\n")
	if endOfLinePos >= 0 {
		res = res[0:endOfLinePos]
	}
	res = strings.Replace(res, `\`, `\\`, -1)
	res = strings.Replace(res, `"`, `\"`, -1)
	res = strings.Replace(res, "\t", `\t`, -1)
	if len(res) > maxSummaryLength {
		res = res[0:maxSummaryLength]
	}
	return res
}

// Clean all Ctrl code from UTF-8 string
// -- Check https://rosettacode.org/wiki/Strip_control_codes_and_extended_characters_from_a_string for more details
//    about this method
// -- This method will remove all C0 control code from string and not the C1 control code. The reason is UTF-8 still
//    allows you to use C1 control characters such as CSI, even though UTF-8 also uses bytes in the range 0x80-0x9F.
// -- For C0 and C1 control code details you can check http://www.cl.cam.ac.uk/~mgk25/unicode.html and
//    https://en.wikipedia.org/wiki/C0_and_C1_control_codes

func stripCtlFromUTF8(str string) string {
	return strings.Map(func(r rune) rune {
		if r >= 32 && r != 127 {
			return r
		}
		return -1
	}, str)
}

// cleanupNewLines removes all newlines from the given input
func cleanupNewLines(s string) string {
	return strings.Replace(strings.Replace(s, "\n", "", -1), "\r", "", -1)
}

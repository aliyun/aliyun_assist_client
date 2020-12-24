package versionutil

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// NOTE: Code below was ported from C++ version of aliyun assistant agent by
// direct translation. Review and refactoring are needed.

type charType int

const (
	_typeNumber charType = iota
	_typePeriod
	_typeString
)

func classifyChar(c rune) charType {
	if c == '.' {
		return _typePeriod
	} else if unicode.IsDigit(c) {
		return _typeNumber
	} else {
		return _typeString
	}
}

// splitVersionString splits version string into individual components. A
// component is continuous run of characters with the same classification. For
// example, "1.20rc3" would be split into ["1",".","20","rc","3"].
func splitVersionString(version string) []string {
	trimedVersion := strings.TrimSpace(version)
	if len(trimedVersion) == 0 {
		return nil
	}

	list := []string{}

	versionRunes := []rune(trimedVersion)
	len := len(versionRunes)

	s := []rune{versionRunes[0]}
	prevType := classifyChar(versionRunes[0])

	for i := 1; i < len; i++ {
		c := versionRunes[i]
		newType := classifyChar(c)

		if prevType != newType || prevType == _typePeriod {
			// We reached a new segment. Period gets special treatment,
			// because "." always delimiters components in version strings
			// (and so ".." means there's empty component value).
			list = append(list, string(s))
			s = []rune{c}
		} else {
			// Add character to current segment and continue.
			s = append(s, c)
		}

		prevType = newType
	}

	// Don't forget to add the last part:
	list = append(list, string(s))

	return list
}

func CompareVersion(verA string, verB string) int {
	partsA := splitVersionString(verA)
	partsB := splitVersionString(verB)

	// Compare common length of both version strings.
	n := len(partsA)
	// Incredible and ridiculous stupidity of google and golang standard library
	// missing min function for int type.
	if len(partsB) < n {
		n = len(partsB)
	}

	for i := 0; i < n; i++ {
		a := partsA[i]
		b := partsB[i]

		firstRuneA, _ := utf8.DecodeLastRuneInString(a)
		typeA := classifyChar(firstRuneA)
		firstRuneB, _ := utf8.DecodeRuneInString(b)
		typeB := classifyChar(firstRuneB)

		if typeA == typeB {
			if typeA == _typeString {
				result := strings.Compare(a, b)
				if result != 0 {
					return result
				}
			} else if typeA == _typeNumber {
				intA, _ := strconv.Atoi(a)
				intB, _ := strconv.Atoi(b)
				if intA > intB  {
					return 1
				} else if intA < intB {
					return -1
				}
			}
		} else { // components of different types
			if typeA != _typeString && typeB == _typeString {
				// 1.2.0 > 1.2rc1
				return 1
			} else if typeA == _typeString && typeB != _typeString {
				// 1.2rc1 < 1.2.0
				return -1
			} else {
				// One is a number and the other is a period. The period
				// is invalid.
				if typeA == _typeNumber {
					return 1
				} else {
					return -1
				}
			}
		}
	}

	// The versions are equal up to the point where they both still have
    // parts. Lets check to see if one is larger than the other.
    if (len(partsA) == len(partsB)) {
		return 0;  // the two strings are identical
	}

               // Lets get the next part of the larger version string
			   // Note that 'n' already holds the index of the part we want.

	var shorterResult, longerResult int
	var missingPartType charType // ('missing' as in "missing in shorter version")

	if len(partsA) > len(partsB) {
		firstRuneAn, _ := utf8.DecodeRuneInString(partsA[n])
		missingPartType = classifyChar(firstRuneAn)
		shorterResult = -1
		longerResult = 1
	} else {
		firstRuneBn, _ := utf8.DecodeRuneInString(partsB[n])
		missingPartType = classifyChar(firstRuneBn)
		shorterResult = 1
		longerResult = -1
	}

	if missingPartType == _typeString {
		// 1.5 > 1.5b3
		return shorterResult
	} else {
		// 1.5.1 > 1.5
		return longerResult
	}
}

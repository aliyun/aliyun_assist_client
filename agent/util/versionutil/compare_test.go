package versionutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyChar(t *testing.T) {
	assert.Exactly(t, _typeNumber, classifyChar('7'),
		"Character '7' should be classified as number type")
	assert.Exactly(t, _typePeriod, classifyChar('.'),
		"Character '.' should be classified as period type")
	assert.Exactly(t, _typeString, classifyChar('g'),
		"Character 'g' should be classified as string type")
}

func TestSplitVersionString(t *testing.T) {
	cases := []struct {
		in string
		expected []string
	}{
		{"1.0.2.589", []string{"1", ".", "0", ".", "2", ".", "589"}},
		{"2.2.0.17", []string{"2", ".", "2", ".", "0", ".", "17"}},
	}

	for _, c := range cases {
		assert.Equalf(t, c.expected, splitVersionString(c.in),
			"Version string %s should be split into %v", c.in, c.expected)
	}
}

func TestCompareVersion(t *testing.T) {
	cases := []struct {
		lhs string
		rhs string
		expected int
	}{
		{"1.0.2.589", "1.0.2.589", 0},
		{"2.2.0.17", "2.2.0.17", 0},
		{"1.0.2.574", "1.0.2.589", -1},
		{"1.0.2.589", "2.2.0.17", -1},
		{"1.0.2.574", "1.0.2.569", 1},
		{"2.2.0.17", "2.2.0.12", 1},

		// Extra cases from examples in implementation code
		{"1.2.0", "1.2rc1", 1},
		{"1.2rc1", "1.2.0", -1},
		{"1.5", "1.5b3", 1},
		{"1.5.1", "1.5", 1},
	}

	for _, c := range cases {
		assert.Exactlyf(t, c.expected, CompareVersion(c.lhs, c.rhs),
			"The result of comparing version string %s and %s should be %d", c.lhs, c.rhs, c.expected)
	}
}

package update

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegexVersionPattern(t *testing.T) {
	regexPattern, err := regexp.Compile(regexVersionPattern)
	assert.NoErrorf(t, err, "Error should be encountered when compling regex pattern %s", regexVersionPattern)

	cases := []struct {
		in string
		expected bool
	}{
		{"1.0.2.589", true},
		{"1", false},
		{"1.0", false},
		{"1.0.2", false},
		{"1.0.2b589", false},
		{"axt1.0.2.589", false},
		{"1.0.2.589commit08b8297c", false},
	}

	for _, c := range cases {
		if c.expected == true {
			assert.Truef(t, regexPattern.MatchString(c.in), "%s should be matched as valid version string", c.in)
		} else {
			assert.Falsef(t, regexPattern.MatchString(c.in), "%s should not be matched as valid version string", c.in)
		}
	}
}

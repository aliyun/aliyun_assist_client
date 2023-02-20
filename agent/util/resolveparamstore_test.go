package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidParameterStore(t *testing.T) {
	references := []string{
		"ss{{oos-secret:test}}",
		"{{ oos-secret : test }}",
		"{{oos-secret: p-a.ra_m}}",
		"{{oos-secret: p-a{{oos-secret:youyong}}ra_m}}",
	}
	for _, reference := range references {
		assert.True(t, isValidParameterStore(reference), reference)
	}
}

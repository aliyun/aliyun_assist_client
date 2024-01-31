package scriptmanager

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeCRLF(t *testing.T) {
	// Single line-terminator style
	assert.Equal(t, "Hello\r\nWorld\r\n",
		NormalizeCRLF("Hello\nWorld\n"))
	assert.Equal(t, "Hello\r\nWorld\r\n",
		NormalizeCRLF("Hello\r\nWorld\r\n"))

	// Mixing line-terminator styles
	assert.Equal(t, "Hello\r\nWorld\r\n",
		NormalizeCRLF("Hello\r\nWorld\n"))
	assert.Equal(t, "Hello\r\nWorld\r\n",
		NormalizeCRLF("Hello\nWorld\r\n"))

	// No line-terminator at the end
	assert.Equal(t, "Hello\r\nWorld",
		NormalizeCRLF("Hello\nWorld"))
	assert.Equal(t, "Hello\r\nWorld",
		NormalizeCRLF("Hello\r\nWorld"))

	// line-terminator at the beginning
	assert.Equal(t, "\r\nHello\r\nWorld",
		NormalizeCRLF("\nHello\nWorld"))
	assert.Equal(t, "\r\nHello\r\nWorld",
		NormalizeCRLF("\r\nHello\r\nWorld"))

	// line-terminator at the beginning, mixing line-terminator styles following
	assert.Equal(t, "\r\nHello\r\nWorld",
		NormalizeCRLF("\nHello\r\nWorld"))
	assert.Equal(t, "\r\nHello\r\nWorld",
		NormalizeCRLF("\r\nHello\nWorld"))

	// Longer texts
	assert.Equal(t, "Hello\r\nWorld\r\nThis is\r\na test\r\n",
		NormalizeCRLF("Hello\nWorld\r\nThis is\na test\n"))

	assert.Equal(t, "Hello\r\nWorld\r\nThis is\r\n\r\n\r\n\r\na test\r\n",
		NormalizeCRLF("Hello\nWorld\r\nThis is\n\n\r\n\na test\n"))

	assert.Equal(t, "\r\nHello\r\nWorld\r\nThis is\r\n\r\n\r\n\r\na test\r\n",
		NormalizeCRLF("\nHello\nWorld\r\nThis is\n\n\r\n\na test\n"))
}

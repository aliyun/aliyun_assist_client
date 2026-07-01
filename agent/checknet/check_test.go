package checknet

import (
	"fmt"
	"testing"
	"math/rand"

	"github.com/stretchr/testify/assert"
)

func TestLimitedBuffer(t *testing.T) {
	testcases := []struct {
		LimitN  int
		WritenN int
	}{
		{
			LimitN: 0,
			WritenN: 0,
		},
		{
			LimitN: 0,
			WritenN: 100,
		},
		{
			LimitN: 99,
			WritenN: 100,
		},
		{
			LimitN: 100,
			WritenN: 100,
		},
		{
			LimitN: 101,
			WritenN: 100,
		},
		{
			LimitN: 110,
			WritenN: 100,
		},
	}

	for _, tc := range testcases {
		t.Run(fmt.Sprintf("limit %d, writen %d", tc.LimitN, tc.WritenN), func(t *testing.T) {
			buf := NewLimitedBuffer(tc.LimitN)
			content := buf.String()
			assert.Equal(t, "", content)

			wcontent := generateRandom(tc.WritenN)
			n, err := buf.Write(wcontent)
			assert.Nil(t, err)
			assert.Equal(t, tc.WritenN, n)

			if tc.LimitN <= tc.WritenN {
				assert.Equal(t, wcontent[:tc.LimitN], []byte(buf.String()))
			} else {
				assert.Equal(t, wcontent, []byte(buf.String()))
			}
			
		})
	}
}

func generateRandom(length int) []byte {
	if length == 0 {
		return []byte{}
	}
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return b
}

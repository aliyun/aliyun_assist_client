package outputbuffer

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRingBuffer(t *testing.T) {
	for quota := 1; quota < 1000; quota += 1 {
		testWithQuota(quota, t)
	}
}

func testWithQuota(quota int, t *testing.T) {
	rb := NewRingBuffer(quota)
	realbuf := []byte{}

	testCases := [][]int{
		{1, 1, 1, 1, 1, 1},
		{3, 3, 3, 3, 3, 3},
		{13, 11, 14, 15, 17},
		{23, 25, 26, 27, 28},
		{53, 57, 51, 59, 50},
		{quota, quota, quota, quota},
	}

	testF := func(n int) {
		content := generateRandom(n)
		// fmt.Println(string(content))
		rb.Put(content)
		// fmt.Printf("quota[%d] len[%d] cap[%d] start[%d] size[%d]\n", rb.quota, len(rb.buf), cap(rb.buf), rb.start, len(rb.buf))
		realbuf = append(realbuf, content...)
		if len(realbuf) < quota {
			if string(rb.Get()) != string(realbuf) {
				fmt.Println("stop: ")
				fmt.Println(string(rb.Get()))
				fmt.Println(string(realbuf))
				panic(1)
			}
			assert.Equal(t, string(rb.Get()), string(realbuf))
		} else {
			if string(rb.Get()) != string(realbuf[len(realbuf)-quota:]) {
				fmt.Println("stop: ")
				fmt.Println(string(rb.Get()))
				fmt.Println(string(realbuf[len(realbuf)-quota:]))
				panic(2)
			}
			assert.Equal(t, string(rb.Get()), string(realbuf[len(realbuf)-quota:]))
		}
	}
	for _, testCase := range testCases {
		// fmt.Println("=================================")
		for _, n := range testCase {
			testF(n)
		}
		rb.Clear()
		realbuf = []byte{}
	}
	for i := 0; i < 100; i += 1 {
		testCase := generateRandomCase(100)
		// fmt.Println("----------------------------------")
		for _, n := range testCase {
			testF(n)
		}
		rb.Clear()
		// fmt.Printf("----- quota[%d] len[%d] cap[%d] start[%d] size[%d]\n", rb.quota, len(rb.buf), cap(rb.buf), rb.start, len(rb.buf))
		realbuf = []byte{}
	}
}

func generateRandomCase(length int) []int {
	b := make([]int, length)
	for i := range b {
		b[i] = rand.Intn(100)
	}
	return b
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

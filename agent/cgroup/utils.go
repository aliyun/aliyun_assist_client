//+build linux

package cgroup

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
)

func writeValue(dir, file, data string) error {
	return ioutil.WriteFile(filepath.Join(dir, file), []byte(data), 0700)
}

func readInt64Value(dir, file string) (int64, error) {
	c, err := ioutil.ReadFile(filepath.Join(dir, file))
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(strings.TrimSpace(string(c)), 10, 64)
}

func parsePairValue(s string) (string, uint64, error) {
	parts := strings.Fields(s)
	switch len(parts) {
	case 2:
		value, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return "", 0, fmt.Errorf("Unable to convert param value (%q) to uint64: %v", parts[1], err)
		}

		return parts[0], value, nil
	default:
		return "", 0, fmt.Errorf("incorrect key-value format: %s", s)
	}
}

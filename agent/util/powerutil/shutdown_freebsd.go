//go:build freebsd
// +build freebsd

package powerutil

import (
	"context"
	"fmt"
)

func IsSystemShutdown(ctx context.Context) (bool, error) {
	return false, fmt.Errorf("unsupport freebsd")
}

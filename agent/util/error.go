package util

import (
	"errors"
)

var (
	ErrRoleNameFailed = errors.New("RoleNameFailed")
	ErrParameterStoreNotAccessible = errors.New("ParameterStoreNotAccessible")
	ErrParameterFailed = errors.New("ParameterFailed")
)

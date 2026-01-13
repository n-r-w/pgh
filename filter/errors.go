package filter

import (
	"errors"
)

var (
	// ErrOrderAliasNotFound SQL alias for orderID was not found.
	ErrOrderAliasNotFound = errors.New("sql alias for orderID not found")

	// ErrUnknownOrderType unknown sort type.
	ErrUnknownOrderType = errors.New("unknown order type")
)

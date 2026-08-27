package couchcmd

import (
	"errors"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func asCapacityExceeded(err error, target **couchcore.CapacityExceededError) bool {
	return errors.As(err, target)
}

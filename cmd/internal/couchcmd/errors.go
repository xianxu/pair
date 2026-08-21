package couchcmd

import (
	"errors"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func asTreeOccupied(err error, target **couchcore.TreeOccupiedError) bool {
	return errors.As(err, target)
}

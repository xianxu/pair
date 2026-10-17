package couchcmd

import (
	"fmt"
	"strconv"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// peerBatchMaxChars validates startup configuration before the broker starts.
func peerBatchMaxChars(getenv func(string) string) (int, error) {
	const key = "COUCH_PEER_BATCH_MAX_CHARS"
	value := getenv(key)
	if value == "" {
		return couchmessage.DefaultPeerBatchMaxChars, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 1 || n > couchmessage.MaxPeerBatchChars {
		return 0, fmt.Errorf("%s must be an integer from 1 through %d", key, couchmessage.MaxPeerBatchChars)
	}
	return n, nil
}

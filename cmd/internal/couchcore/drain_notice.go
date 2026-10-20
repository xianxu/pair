package couchcore

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"unicode"
	"unicode/utf8"
)

var drainUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// DrainNotice is the single owner of the shared v1 worker protocol text.
func DrainNotice(id, reason string, fast bool) (string, error) {
	if !drainUUID.MatchString(id) {
		return "", errors.New("drain ID must be a canonical UUID")
	}
	if !utf8.ValidString(reason) || len(reason) > 1024 {
		return "", errors.New("drain reason must be valid UTF-8 of at most 1024 bytes")
	}
	for _, r := range reason {
		if unicode.IsControl(r) {
			return "", errors.New("drain reason contains terminal controls")
		}
	}
	var text string
	if fast {
		text = fmt.Sprintf("Couch drain v1 %s (fast): stop now and checkpoint. Do not finish the current step. Write where you stopped in the issue Log; commit and push the checkpoint without discarding unfinished working-tree edits. Start nothing new. Run couch --drain --ack %s after the checkpoint is durable, then end your turn and stay idle. Do not send a separate drained report.", id, id)
	} else {
		text = fmt.Sprintf("Couch drain v1 %s: checkpoint and stop. Finish the current atomic step; commit and push the checkpoint; write where you stopped in the issue Log. Start nothing new. Run couch --drain --ack %s after the checkpoint is durable, then end your turn and stay idle. Do not send a separate drained report.", id, id)
	}
	if reason != "" {
		text += "\nReason: " + strconv.Quote(reason)
	}
	return text, nil
}

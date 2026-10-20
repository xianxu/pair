package couchcore

import (
	"errors"
	"strconv"
	"unicode"
	"unicode/utf8"
)

// DrainNotice is ordinary guidance to the worker, not a control protocol.
func DrainNotice(fast bool) string {
	text := "Drain: finish your current atomic step and wait for current background work to finish."
	if fast {
		text = "Drain (fast): stop now; do not finish the current step. Promptly stop your own background work via supported controls; preserve unfinished working-tree edits in your checkpoint."
	}
	return text + " Write where you stopped and the next step in the issue Log; commit and push the checkpoint, then end your turn and stay idle. Start no new work. While idle, keep receiving messages. For new work, reply once: 'acknowledge message received, will resume processing after drain is lifted'. Do not create courtesy acknowledgements or acknowledgement loops."
}

// UndrainNotice treats the optional operator note as quoted data.
func UndrainNotice(message string) (string, error) {
	if !utf8.ValidString(message) || len(message) > 1024 {
		return "", errors.New("undrain message must be valid UTF-8 of at most 1024 bytes")
	}
	for _, r := range message {
		if unicode.IsControl(r) {
			return "", errors.New("undrain message contains terminal controls")
		}
	}
	text := "drain lifted; continue your current work"
	if message != "" {
		text += "\nNote: " + strconv.Quote(message)
	}
	return text, nil
}

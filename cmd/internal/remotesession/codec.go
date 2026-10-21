package remotesession

import (
	"errors"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

func validateRemoteJSON(raw []byte, maxBytes int) error {
	if len(raw) > maxBytes {
		return errors.New("remote record exceeds size limit")
	}
	if err := strictjson.RejectDuplicateKeysPreserveNumbers(raw); err != nil {
		return err
	}
	// Count only structural delimiters outside strings. Syntax was checked above.
	depth := 0
	quoted, escaped := false, false
	for _, b := range raw {
		if quoted {
			if escaped {
				escaped = false
			} else if b == '\\' {
				escaped = true
			} else if b == '"' {
				quoted = false
			}
			continue
		}
		switch b {
		case '"':
			quoted = true
		case '{', '[':
			depth++
			if depth > 32 {
				return errors.New("remote record exceeds depth limit")
			}
		case '}', ']':
			depth--
		}
	}

	return nil
}

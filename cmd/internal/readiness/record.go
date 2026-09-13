package readiness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/orientation"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

type ReadyRecord struct {
	Orientation *orientation.DeliveryState `json:"orientation,omitempty"`
	Tag         string                     `json:"tag"`
	Agent       string                     `json:"agent"`
	Session     string                     `json:"session"`
	Nonce       string                     `json:"nonce"`
	PID         int                        `json:"pid"`
}

func Encode(record ReadyRecord) (string, error) {
	if err := Validate(record); err != nil {
		return "", err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(record); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func Decode(raw string) (ReadyRecord, error) {
	var record ReadyRecord
	if err := strictjson.Decode([]byte(raw), &record); err != nil {
		return ReadyRecord{}, err
	}
	if err := Validate(record); err != nil {
		return ReadyRecord{}, err
	}
	return record, nil
}

func Validate(record ReadyRecord) error {
	if s := record.Orientation; s != nil {
		if len(s.Reason) > 1024 || !utf8.ValidString(s.Reason) || strings.IndexFunc(s.Reason, unicode.IsControl) >= 0 {
			return fmt.Errorf("ready record: invalid orientation reason")
		}
		switch s.Phase {
		case orientation.DeliveryWaiting, orientation.DeliveryPasting, orientation.DeliveryFailed:
			if s.BodyWritten {
				return fmt.Errorf("ready record: unexpected orientation body")
			}
		case orientation.DeliverySettling, orientation.DeliverySubmitting, orientation.DeliverySubmitted, orientation.DeliveryIndeterminate:
			if !s.BodyWritten {
				return fmt.Errorf("ready record: missing orientation body")
			}
		case orientation.DeliveryCancelled:
		default:
			return fmt.Errorf("ready record: unknown orientation phase")
		}
	}

	switch {
	case record.Tag == "":
		return fmt.Errorf("ready record: tag is empty")
	case record.Agent == "":
		return fmt.Errorf("ready record: agent is empty")
	case record.Session == "":
		return fmt.Errorf("ready record: session is empty")
	case record.Nonce == "":
		return fmt.Errorf("ready record: nonce is empty")
	case record.PID <= 0:
		return fmt.Errorf("ready record: pid must be positive")
	default:
		return nil
	}
}

package readiness

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type ReadyRecord struct {
	Tag     string `json:"tag"`
	Agent   string `json:"agent"`
	Session string `json:"session"`
	Nonce   string `json:"nonce"`
	PID     int    `json:"pid"`
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
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return ReadyRecord{}, err
	}
	if err := Validate(record); err != nil {
		return ReadyRecord{}, err
	}
	return record, nil
}

func Validate(record ReadyRecord) error {
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

package sessionwatch

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/xianxu/pair/cmd/internal/nativeid"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

const BindingNoticeEnv = "PAIR_BINDING_NOTICE_FD"
const MaxBindingNotice = 1024

type bindingNotice struct {
	Ordinal  uint64 `json:"ordinal"`
	NativeID string `json:"native_id"`
}

func validBindingNotice(n bindingNotice) bool {
	return n.Ordinal != 0 && nativeid.Validate(n.NativeID) == nil
}

// NotifyBinding only transfers an already-committed current-launch decision.
// Failure is visible to the wrapper as an incomplete discovery channel, never
// permission to retry native input or to invent an ID.
func NotifyBinding(w io.Writer, ordinal uint64, id string) error {
	n := bindingNotice{ordinal, id}
	if !validBindingNotice(n) {
		return errors.New("invalid binding notification")
	}
	return json.NewEncoder(w).Encode(n)
}

func DecodeBindingNotice(raw []byte, ordinal uint64) (string, error) {
	var n bindingNotice
	if len(raw) > MaxBindingNotice {
		return "", errors.New("binding notification too large")
	}
	if err := strictjson.Decode(raw, &n); err != nil {
		return "", err
	}
	if !validBindingNotice(n) || n.Ordinal != ordinal {
		return "", errors.New("binding notification is not for this launch")
	}
	return n.NativeID, nil
}

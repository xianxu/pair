package couchtty

import (
	"bytes"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

// CopyKeyObservation reports only Ctrl+Y arrival, never input bytes or text.
type CopyKeyObservation struct {
	Encoding string `json:"encoding"`
	Panel    bool   `json:"panel"`
	Release  bool   `json:"release"`
}

// SetCopyKeyObserver installs the content-free arrival diagnostic sink.
func (c *Console) SetCopyKeyObserver(observer func(CopyKeyObservation)) {
	c.mu.Lock()
	c.copyKeyObserver = observer
	c.mu.Unlock()
}

func copyKeyObservation(event terminal.InputEvent) (CopyKeyObservation, bool) {
	var key uv.Key
	release := false
	switch e := event.Event.(type) {
	case uv.KeyPressEvent:
		key = uv.Key(e)
	case uv.KeyReleaseEvent:
		key = uv.Key(e)
		release = true
	default:
		return CopyKeyObservation{}, false
	}
	mod := key.Mod &^ (uv.ModCapsLock | uv.ModNumLock | uv.ModScrollLock)
	if key.Code != 'y' || mod != uv.ModCtrl {
		return CopyKeyObservation{}, false
	}
	encoding := "enhanced"
	if bytes.Equal(event.Raw, []byte{0x19}) {
		encoding = "legacy"
	}
	return CopyKeyObservation{Encoding: encoding, Release: release}, true
}

func (c *Console) observeCopyKey(event terminal.InputEvent) {
	observation, ok := copyKeyObservation(event)
	if !ok {
		return
	}
	c.mu.Lock()
	observation.Panel = c.focus.IsPanel()
	observer := c.copyKeyObserver
	c.mu.Unlock()
	if observer != nil {
		observer(observation)
	}
}

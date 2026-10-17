package couchcmd

import "github.com/xianxu/pair/cmd/internal/couchtty"

func clipboardCopyFact(observation couchtty.CopyObservation) supervisionFact {
	return supervisionFact{Event: "clipboard_copy", Clipboard: &observation}
}

func (l *supervisionFactLog) clipboardCopy(observation couchtty.CopyObservation) {
	l.offer(clipboardCopyFact(observation))
}

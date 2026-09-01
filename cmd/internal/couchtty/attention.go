package couchtty

import (
	"math"
	"sort"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

const attentionPerActor = 3

type AttentionMessage struct {
	Sequence uint64
	Text     string
}

type AttentionCapture uint64

type attentionCaptureRecord struct {
	address   couchcore.ThreadAddress
	sequences []uint64
}

// AttentionLedger is the sole ephemeral authority for which actors need the
// operator. Its values are pure in-memory observations; no notification is
// persisted into Couch's thread store.
type AttentionLedger struct {
	records      map[couchcore.ThreadAddress][]AttentionMessage
	nextSequence uint64
	captures     map[AttentionCapture]attentionCaptureRecord
	nextCapture  AttentionCapture
}

func (l *AttentionLedger) Mark(address couchcore.ThreadAddress, text string) {
	l.init()
	if l.nextSequence == math.MaxUint64 {
		l.rebase()
	}
	l.nextSequence++
	messages := l.records[address]
	for i := 0; i < len(messages); i++ {
		if messages[i].Text == text {
			messages = append(messages[:i], messages[i+1:]...)
			break
		}
	}
	messages = append(messages, AttentionMessage{Sequence: l.nextSequence, Text: text})
	if len(messages) > attentionPerActor {
		messages = append([]AttentionMessage(nil), messages[len(messages)-attentionPerActor:]...)
	}
	l.records[address] = messages
}

func (l *AttentionLedger) Projection(address couchcore.ThreadAddress) []AttentionMessage {
	if l == nil {
		return nil
	}
	return append([]AttentionMessage(nil), l.records[address]...)
}

func (l *AttentionLedger) NewestActor() couchcore.ThreadAddress {
	var newest couchcore.ThreadAddress
	var sequence uint64
	for address, messages := range l.records {
		for _, message := range messages {
			if message.Sequence > sequence {
				sequence = message.Sequence
				newest = address
			}
		}
	}
	return newest
}

func (l *AttentionLedger) Capture(address couchcore.ThreadAddress) AttentionCapture {
	l.init()
	messages := l.records[address]
	if len(messages) == 0 {
		return 0
	}
	l.nextCapture++
	if l.nextCapture == 0 {
		l.nextCapture++
	}
	sequences := make([]uint64, len(messages))
	for i := range messages {
		sequences[i] = messages[i].Sequence
	}
	l.captures[l.nextCapture] = attentionCaptureRecord{address: address, sequences: sequences}
	return l.nextCapture
}

func (l *AttentionLedger) Acknowledge(capture AttentionCapture) {
	if l == nil || capture == 0 {
		return
	}
	record, ok := l.captures[capture]
	if !ok {
		return
	}
	delete(l.captures, capture)
	wanted := make(map[uint64]struct{}, len(record.sequences))
	for _, sequence := range record.sequences {
		wanted[sequence] = struct{}{}
	}
	messages := l.records[record.address]
	kept := messages[:0]
	for _, message := range messages {
		if _, clear := wanted[message.Sequence]; !clear {
			kept = append(kept, message)
		}
	}
	if len(kept) == 0 {
		delete(l.records, record.address)
	} else {
		l.records[record.address] = kept
	}
}

func (l *AttentionLedger) Cancel(capture AttentionCapture) {
	if l != nil {
		delete(l.captures, capture)
	}
}

func (l *AttentionLedger) DropActor(address couchcore.ThreadAddress) {
	if l == nil {
		return
	}
	delete(l.records, address)
	for capture, record := range l.captures {
		if record.address == address {
			delete(l.captures, capture)
		}
	}
}

func (l *AttentionLedger) init() {
	if l.records == nil {
		l.records = make(map[couchcore.ThreadAddress][]AttentionMessage)
	}
	if l.captures == nil {
		l.captures = make(map[AttentionCapture]attentionCaptureRecord)
	}
}

func (l *AttentionLedger) rebase() {
	type position struct {
		address couchcore.ThreadAddress
		index   int
		old     uint64
	}
	positions := make([]position, 0)
	for address, messages := range l.records {
		for i, message := range messages {
			positions = append(positions, position{address: address, index: i, old: message.Sequence})
		}
	}
	sort.Slice(positions, func(i, j int) bool { return positions[i].old < positions[j].old })
	rebased := make(map[uint64]uint64, len(positions))
	for i, position := range positions {
		sequence := uint64(i + 1)
		messages := l.records[position.address]
		messages[position.index].Sequence = sequence
		l.records[position.address] = messages
		rebased[position.old] = sequence
	}
	for capture, record := range l.captures {
		for i, sequence := range record.sequences {
			record.sequences[i] = rebased[sequence]
		}
		l.captures[capture] = record
	}
	l.nextSequence = uint64(len(positions))
}

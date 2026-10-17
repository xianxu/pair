package couchmessage

import (
	"strings"
	"testing"
)

func TestPeerBatchValidationRejectsBeyondQualifiedComposerRange(t *testing.T) {
	head := batchFixture("head", strings.Repeat("a", MaxBodyBytes))
	members := []Message{head}
	for _, id := range []string{"two", "three", "four", "five"} {
		m := head
		m.ID = id
		members = append(members, m)
	}
	if !batchFinalizationFits(members) {
		t.Fatal("fixture must fit the transport independently of composer cap")
	}
	if err := ValidateBatch(members, head.To, head.ID); err == nil {
		t.Fatal("accepted untrusted batch beyond qualified composer range")
	}
}

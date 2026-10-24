package wrapcmd

import (
	"bytes"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/agentdrive"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestAgentDriveCodexTextQuietBeforeEnter(t *testing.T) {
	for _, kind := range []agentdrive.KeyKind{agentdrive.KeyEnter, agentdrive.KeySubmit} {
		t.Run(string(kind), func(t *testing.T) {
			f := newHarnessSessionFake(t, "codex", true)
			defer f.close()
			now := time.Now()
			d := newPeerDelivery(couchmessage.Binding{Agent: "codex"}, func() time.Time { return now })
			f.proxy.peer = d
			d.driveProbe = f.proxy.agentDriveScreenLocked
			d.settled = true
			f.output("\x1b[?2026h\x1b[2J\x1b[20;1H\x1b[1m›\x1b[22m \x1b[?25h\x1b[20;3H\x1b[?2026l")
			result, err := driveStart(t, d, agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyText, Text: "/model"}, {Kind: kind}}})
			if err != nil || result.Terminal() {
				t.Fatalf("admission %+v %v", result, err)
			}
			var out bytes.Buffer
			f.proxy.dispatchAgentDrive(&out)
			if out.String() != "/model" {
				t.Fatalf("text %q", out.String())
			}
			f.output("\x1b[?2026h\x1b[2J" + slashComposerPaint("codex", "/model") + "\x1b[?2026l")
			f.proxy.dispatchAgentDrive(&out) // echo proof does not waive quiet time
			now = now.Add(PeerSubmitDelay - time.Nanosecond)
			f.proxy.dispatchAgentDrive(&out)
			if out.String() != "/model" {
				t.Fatalf("Enter reached Codex during its paste suppression window: %q", out.String())
			}
			now = now.Add(time.Nanosecond)
			f.proxy.dispatchAgentDrive(&out)
			if out.Len() != len("/model")+1 {
				t.Fatalf("quiet period did not release exactly one Enter: %q", out.String())
			}
		})
	}
}

func TestAgentDriveCodexTextQuietCancellation(t *testing.T) {
	f, d, now := driveFixture(t)
	// Keep the fixture's qualified composer while selecting Codex's receiver
	// pacing policy. The shared writer must still honor human cancellation.
	d.binding.Agent = "codex"
	_, err := driveStart(t, d, agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyText, Text: "/model"}, {Kind: agentdrive.KeySubmit}}})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	f.proxy.dispatchAgentDrive(&out)
	f.proxy.admitStdin([]byte("operator"), false)
	*now = now.Add(PeerSubmitDelay)
	f.proxy.dispatchAgentDrive(&out)
	if out.String() != "/model" {
		t.Fatalf("pending Enter survived cancellation: %q", out.String())
	}
}

func TestAgentDriveCodexQuietCannotConsumeRecoveryReserve(t *testing.T) {
	f, d, now := driveFixture(t)
	d.binding.Agent = "codex"
	_, err := driveStart(t, d, agentdrive.Request{Model: &agentdrive.ModelRequest{Model: "gpt-6.1-sol"}})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	f.proxy.dispatchAgentDrive(&out) // clear-line
	*now = now.Add(850 * time.Millisecond)
	f.proxy.dispatchAgentDrive(&out) // text acknowledged close to the deadline
	recoveryPaint(f, "/model")
	before := out.Len()
	*now = now.Add(50 * time.Millisecond) // reserved cleanup starts at 900ms
	f.proxy.dispatchAgentDrive(&out)
	if out.Len() != before+1 || out.Bytes()[out.Len()-1] != 0x15 {
		t.Fatalf("pending submit consumed cleanup reserve: %q", out.Bytes()[before:])
	}
	f.output("\x1b[?2026h\x1b[2J" + peerEmptyComposer() + "\x1b[?2026l")
	f.proxy.dispatchAgentDrive(&out)
	result, _ := d.agentDriveResult("test-drive")
	if result.Outcome != agentdrive.OutcomeFailed {
		t.Fatalf("fresh recovery not established: %+v", result)
	}
}

func TestAgentDriveCodexPendingEnterHonorsAbsoluteClock(t *testing.T) {
	f, d, now := driveFixture(t)
	d.binding.Agent = "codex"
	_, err := driveStart(t, d, agentdrive.Request{Keys: agentdrive.KeySequence{{Kind: agentdrive.KeyText, Text: "/model"}, {Kind: agentdrive.KeySubmit}}})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	f.proxy.dispatchAgentDrive(&out)
	*now = now.Add(time.Second) // no reliance on the real context timer callback
	f.proxy.dispatchAgentDrive(&out)
	if out.String() != "/model" {
		t.Fatalf("Enter emitted at absolute deadline: %q", out.String())
	}
}

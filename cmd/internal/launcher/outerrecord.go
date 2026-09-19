package launcher

import (
	"fmt"
	"strings"
)

// OuterRecord is what an attaching Pair client writes about its outer terminal
// (outer-tty-<tag>): the tty, and whether Couch is the one presenting it.
// Written on every create and attach, so the last attach wins (#282).
type OuterRecord struct {
	TTY   string
	Couch bool
}

const (
	presenterCouch    = "presenter=couch"
	presenterTerminal = "presenter=terminal"
)

// EncodeOuterRecord renders the record: the tty line, then the presenter line.
func EncodeOuterRecord(r OuterRecord) string {
	presenter := presenterTerminal
	if r.Couch {
		presenter = presenterCouch
	}
	return r.TTY + "\n" + presenter + "\n"
}

// DecodeOuterRecord is strict. A one-line record predates #282 and says
// nothing about Couch; every other shape is refused rather than guessed, so a
// truncated or garbled record degrades visibly instead of reading as Couch.
func DecodeOuterRecord(raw string) (OuterRecord, error) {
	lines := strings.Split(strings.TrimSuffix(raw, "\n"), "\n")
	if !strings.HasPrefix(lines[0], "/dev/") {
		return OuterRecord{}, fmt.Errorf("outer-tty record: bad tty line %q", lines[0])
	}
	switch {
	case len(lines) == 1:
		return OuterRecord{TTY: lines[0]}, nil
	case len(lines) == 2 && lines[1] == presenterCouch:
		return OuterRecord{TTY: lines[0], Couch: true}, nil
	case len(lines) == 2 && lines[1] == presenterTerminal:
		return OuterRecord{TTY: lines[0]}, nil
	}
	return OuterRecord{}, fmt.Errorf("outer-tty record: unexpected shape (%d lines)", len(lines))
}

// PresentedByCouch reports whether Couch launched THIS client for THIS thread.
// The client's env is set per attach by whoever launched it -- unlike the
// Zellij server's env, which records who created the session (#282). The tag
// must match, so a COUCH_THREAD_TAG inherited by some other launch is not
// mistaken for Couch.
func PresentedByCouch(env Env, tag string) bool {
	return env.CouchThreadTag != "" && env.CouchThreadTag == tag
}

// CouchOwnsRestart is the rule for whether Pair may restart a session in place
// (#284): not when the session env names Couch (Couch created it), and not when
// Couch presents the attached client, whose launcher refuses the restart marker
// -- but only after quit cleanup has already torn the thread down.
//
// `pair keys` reads the same rule for its hosted wording. The two consumers
// agree on every answer the predicate gives, and DIVERGE where it cannot be
// computed: reading the presenter record can fail, and a help page that cannot
// render is worse than a slightly wrong row, while a restart that cannot be
// judged is worse than no restart. So the help fails open to Pair's own page
// (keyscmd TestUnreadablePresenterRendersPairsPage) and the gate fails closed
// (TestRunRestartRefusesACouchOwnedSessionBeforeMutation's unreadable case).
// On that one input the help can promise a reload the gate then refuses, which
// the refusal says out loud.
func CouchOwnsRestart(sessionEnvHosted, presentedByCouch bool) bool {
	return sessionEnvHosted || presentedByCouch
}

// couchRestartGate runs before an in-session restart-marker writer writes
// anything (#284). The presenter record is input this process did not write, and
// it gates killing a live session, so a record that cannot be read refuses.
//
// An unresolved tag is the one "cannot tell" that proceeds, and not because it
// is less uncertain: it is a state Couch cannot be in. The record is keyed by
// tag, so there is nothing to read -- but reaching it needs PAIR_TAG unset AND
// the session name absent from the index, while every session the launcher
// creates exports PAIR_TAG into its panes, and every session Couch creates also
// carries COUCH_THREAD_* in the session env, which the env half above catches
// without the record. What is left is a pre-index legacy session, which no
// Couch ever created or presented. Refusing there would cost that session its
// restart to guard a state it cannot reach.
func couchRestartGate(rt Runtime, sessionEnvHosted bool, tag string) error {
	presented := false
	if !sessionEnvHosted && tag != "" {
		var err error
		if presented, err = rt.OuterPresenter(tag); err != nil {
			return fmt.Errorf("cannot tell whether Couch presents this session, so it is not restarted in place: %v; re-attaching with `pair` rewrites that record", err)
		}
	}
	if CouchOwnsRestart(sessionEnvHosted, presented) {
		return fmt.Errorf("this session's restarts belong to Couch; relaunch the thread from Couch (Alt+n)")
	}
	return nil
}

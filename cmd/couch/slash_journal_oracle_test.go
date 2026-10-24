package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
)

// slashJournalOracle is test-only authority for an append-only accepted-input
// journal. A display header supplies a lookup candidate; typed broker proof is
// the authority for every system notice appended after the immutable baseline.
type slashJournalOracle struct {
	baseline    []agentfixture.AcceptedInput
	observed    []agentfixture.AcceptedInput
	hub, worker couchmessage.Binding
}

func newSlashJournalOracle(baseline []agentfixture.AcceptedInput, hub, worker couchmessage.Binding) *slashJournalOracle {
	return &slashJournalOracle{
		baseline: append([]agentfixture.AcceptedInput(nil), baseline...),
		observed: append([]agentfixture.AcceptedInput(nil), baseline...),
		hub:      hub, worker: worker,
	}
}

func (o *slashJournalOracle) Observe(current []agentfixture.AcceptedInput, proofs map[string]couchmessage.Receipt) (bool, error) {
	if len(current) > agentfixture.MaxRequests || len(o.baseline) > agentfixture.MaxRequests {
		return false, fmt.Errorf("accepted journal exceeds %d entries", agentfixture.MaxRequests)
	}
	if len(current) < len(o.observed) {
		return false, fmt.Errorf("accepted journal truncated from %d to %d entries", len(o.observed), len(current))
	}
	for i, old := range o.observed {
		if current[i] != old {
			return false, fmt.Errorf("accepted journal prefix changed at entry %d", i)
		}
	}
	seen := make(map[string]bool, len(current))
	for _, entry := range o.baseline {
		if id, ok := systemEnvelopeID(entry.Body); ok {
			seen[id] = true
		}
	}
	commands := 0
	pending := false
	for i, entry := range current[len(o.baseline):] {
		if entry.Body == "/model" {
			commands++
			if commands > 1 {
				return false, fmt.Errorf("duplicate /model acceptance after baseline")
			}
			continue
		}
		id, ok := systemEnvelopeID(entry.Body)
		if !ok {
			return false, fmt.Errorf("unknown appended input at entry %d", len(o.baseline)+i)
		}
		if seen[id] {
			return false, fmt.Errorf("duplicate system delivery %q", id)
		}
		seen[id] = true
		proof, ok := proofs[id]
		if !ok {
			return false, fmt.Errorf("missing broker proof for system delivery %q", id)
		}
		submitted, err := slashSystemProofSubmitted(entry, proof, o.hub, o.worker)
		if err != nil {
			return false, err
		}
		pending = pending || !submitted
	}
	// Even pending observations pin the complete prefix, without retaining aliases
	// to a caller's journal slice or accepting mutations in an errored observation.
	o.observed = append([]agentfixture.AcceptedInput(nil), current...)
	return commands == 1 && !pending, nil
}

// systemEnvelopeID parses only the first exact system header. Its result is a
// lookup candidate, never authority to ignore an arbitrary body or a batch.
func systemEnvelopeID(body string) (string, bool) {
	header, _, ok := strings.Cut(body, "\n")
	if !ok {
		return "", false
	}
	id, ok := strings.CutPrefix(header, "[Couch system; delivery ")
	if !ok {
		return "", false
	}
	id, ok = strings.CutSuffix(id, "]")
	if !ok || id == "" || len(id) > 128 || !utf8.ValidString(id) || strings.ContainsAny(id, "[]") || strings.IndexFunc(id, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
		return "", false
	}
	return id, true
}

// slashSystemProofSubmitted binds the entire observed envelope to the broker's
// typed system origin and complete endpoint identities before trusting status.
func slashSystemProofSubmitted(entry agentfixture.AcceptedInput, proof couchmessage.Receipt, hub, worker couchmessage.Binding) (bool, error) {
	id, ok := systemEnvelopeID(entry.Body)
	if !ok || proof.Message.ID != id || proof.Message.Origin != couchmessage.OriginSystem || proof.Message.From != hub || proof.Message.To != worker || entry.Body != couchmessage.Envelope(proof.Message) {
		return false, fmt.Errorf("invalid broker proof for system delivery %q", id)
	}
	switch proof.Status {
	case couchmessage.Submitted:
		return true, nil
	case couchmessage.Queued, couchmessage.Held, couchmessage.Delivering:
		return false, nil
	default:
		return false, fmt.Errorf("system delivery %q has unacceptable broker status %q", id, proof.Status)
	}
}

func slashOracleBindings() (couchmessage.Binding, couchmessage.Binding) {
	hub := couchmessage.Binding{Slot: "fixture:0", Repository: "/fixture/.git", Scope: "scope", Tag: "hub", Session: "session-hub", Nonce: "hub-nonce", Agent: "claude", Version: "1", PID: 10, Start: "hub-start"}
	worker := hub
	worker.Slot = "fixture:1"
	worker.Tag = "worker"
	worker.Session = "session-worker"
	worker.Nonce = "worker-nonce"
	worker.PID = 11
	worker.Start = "worker-start"
	return hub, worker
}
func slashOracleInput(seq uint64, body string) agentfixture.AcceptedInput {
	return agentfixture.AcceptedInput{PID: 11, Sequence: seq, NativeID: "native-id", Body: body, At: time.Unix(int64(seq), 0)}
}
func slashOracleReceipt(id string, status couchmessage.Status) couchmessage.Receipt {
	hub, worker := slashOracleBindings()
	return couchmessage.Receipt{Message: couchmessage.Message{ID: id, Origin: couchmessage.OriginSystem, From: hub, To: worker, Body: "Slots update\nfixture:1 ready"}, Status: status}
}

func TestSlashJournalOracleAllowsProvedSystemNotice(t *testing.T) {
	hub, worker := slashOracleBindings()
	baseline := []agentfixture.AcceptedInput{slashOracleInput(1, "startup")}
	receipt := slashOracleReceipt("notice-1", couchmessage.Submitted)
	current := append(append([]agentfixture.AcceptedInput(nil), baseline...), slashOracleInput(2, couchmessage.Envelope(receipt.Message)), slashOracleInput(3, "/model"))
	ready, err := newSlashJournalOracle(baseline, hub, worker).Observe(current, map[string]couchmessage.Receipt{"notice-1": receipt})
	if err != nil || !ready {
		t.Fatalf("proved notice blocked command: ready=%v err=%v", ready, err)
	}
}
func TestSlashJournalOraclePinsPendingPrefix(t *testing.T) {
	hub, worker := slashOracleBindings()
	baseline := []agentfixture.AcceptedInput{slashOracleInput(1, "startup")}
	oracle := newSlashJournalOracle(baseline, hub, worker)
	if ready, err := oracle.Observe(baseline, nil); err != nil || ready {
		t.Fatalf("baseline: %v %v", ready, err)
	}
	changed := append([]agentfixture.AcceptedInput(nil), baseline...)
	changed[0].PID++
	if _, err := oracle.Observe(changed, nil); err == nil {
		t.Fatal("same-length prefix mutation accepted")
	}
}

func TestSystemEnvelopeIDRequiresExactFirstHeader(t *testing.T) {
	for _, tc := range []struct{ body, id string }{
		{"[Couch system; delivery notice-1]\nSlots update", "notice-1"},
		{"[Couch system; delivery notice_2]\n", "notice_2"},
		{"prefix\n[Couch system; delivery notice-1]\nSlots update", ""},
		{" [Couch system; delivery notice-1]\nSlots update", ""},
		{"[Couch peer from fixture:0; delivery notice-1]\nSlots update", ""},
		{"[Couch system; delivery ]\nSlots update", ""},
		{"[Couch system; delivery notice 1]\nSlots update", ""},
		{"[Couch system; delivery notice\t1]\nSlots update", ""},
		{"[Couch system; delivery notice-1]\r\nSlots update", ""},
		{"[Couch system; delivery notice-1]", ""},
		{"[Couch system; delivery notice-1]extra\nSlots update", ""},
		{"[Couch system; delivery [notice-1]]\nSlots update", ""},
		{"[Couch system; delivery " + strings.Repeat("x", 129) + "]\nSlots update", ""},
		{"[Couch system; delivery bad\xff]\nSlots update", ""},
	} {
		t.Run(tc.body, func(t *testing.T) {
			id, ok := systemEnvelopeID(tc.body)
			if id != tc.id || ok != (tc.id != "") {
				t.Fatalf("candidate=%q,%v want %q", id, ok, tc.id)
			}
		})
	}
}

func TestSlashSystemProofSubmittedStatuses(t *testing.T) {
	hub, worker := slashOracleBindings()
	for _, status := range []couchmessage.Status{couchmessage.Queued, couchmessage.Held, couchmessage.Delivering, couchmessage.Submitted, couchmessage.Expired, couchmessage.Cancelled, couchmessage.Indeterminate, couchmessage.Refused, "", "unknown"} {
		t.Run(string(status), func(t *testing.T) {
			proof := slashOracleReceipt("notice-1", status)
			entry := slashOracleInput(1, couchmessage.Envelope(proof.Message))
			submitted, err := slashSystemProofSubmitted(entry, proof, hub, worker)
			pending := status == couchmessage.Queued || status == couchmessage.Held || status == couchmessage.Delivering
			if status == couchmessage.Submitted {
				if err != nil || !submitted {
					t.Fatalf("submitted: %v %v", submitted, err)
				}
			} else if pending {
				if err != nil || submitted {
					t.Fatalf("pending: %v %v", submitted, err)
				}
			} else if err == nil || submitted {
				t.Fatalf("terminal/unknown failure trusted: %v %v", submitted, err)
			}
		})
	}
}

func TestSlashSystemProofSubmittedBindsAllIdentityFields(t *testing.T) {
	hub, worker := slashOracleBindings()
	original := slashOracleReceipt("notice-1", couchmessage.Submitted)
	entry := slashOracleInput(1, couchmessage.Envelope(original.Message))
	for _, endpoint := range []string{"From", "To"} {
		typ := reflect.TypeOf(hub)
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i).Name
			t.Run(endpoint+"/"+field, func(t *testing.T) {
				proof := original
				identity := &proof.Message.From
				if endpoint == "To" {
					identity = &proof.Message.To
				}
				value := reflect.ValueOf(identity).Elem().FieldByName(field)
				if value.Kind() == reflect.String {
					value.SetString(value.String() + "-changed")
				} else {
					value.SetInt(value.Int() + 1)
				}
				if submitted, err := slashSystemProofSubmitted(entry, proof, hub, worker); err == nil || submitted {
					t.Fatalf("changed %s.%s trusted", endpoint, field)
				}
			})
		}
	}
	for _, tc := range []struct {
		name   string
		mutate func(*couchmessage.Receipt)
	}{
		{"receipt-id", func(p *couchmessage.Receipt) { p.Message.ID = "other" }},
		{"peer-origin", func(p *couchmessage.Receipt) { p.Message.Origin = couchmessage.OriginPeer }},
		{"unknown-origin", func(p *couchmessage.Receipt) { p.Message.Origin = "spoof" }},
		{"receipt-body", func(p *couchmessage.Receipt) { p.Message.Body += "changed" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			proof := original
			tc.mutate(&proof)
			if ok, err := slashSystemProofSubmitted(entry, proof, hub, worker); err == nil || ok {
				t.Fatal("spoofed proof accepted")
			}
		})
	}
	for _, body := range []string{
		entry.Body + "\nextra", "prose", "[Couch system; delivery notice-1]\nspoofed notice",
		couchmessage.BatchEnvelope([]couchmessage.Message{original.Message, slashOracleReceipt("notice-2", couchmessage.Submitted).Message}),
	} {
		t.Run(body, func(t *testing.T) {
			changed := entry
			changed.Body = body
			if ok, err := slashSystemProofSubmitted(changed, original, hub, worker); err == nil || ok {
				t.Fatal("altered/batched envelope accepted")
			}
		})
	}
}

func TestSlashJournalOracleStatefulSystemTransitions(t *testing.T) {
	hub, worker := slashOracleBindings()
	before := slashOracleReceipt("before-baseline", couchmessage.Submitted)
	baseline := []agentfixture.AcceptedInput{slashOracleInput(1, couchmessage.Envelope(before.Message)), slashOracleInput(2, "/model"), slashOracleInput(3, "startup prose")}
	oracle := newSlashJournalOracle(baseline, hub, worker)
	current := append([]agentfixture.AcceptedInput(nil), baseline...)
	proofs := map[string]couchmessage.Receipt{}
	check := func(want bool) {
		t.Helper()
		ready, err := oracle.Observe(current, proofs)
		if err != nil || ready != want {
			t.Fatalf("ready=%v want=%v err=%v", ready, want, err)
		}
	}
	check(false)
	notice := slashOracleReceipt("before-command", couchmessage.Queued)
	current = append(current, slashOracleInput(4, couchmessage.Envelope(notice.Message)))
	for _, status := range []couchmessage.Status{couchmessage.Queued, couchmessage.Held, couchmessage.Delivering} {
		notice.Status = status
		proofs[notice.Message.ID] = notice
		check(false)
	}
	current = append(current, slashOracleInput(5, "/model"))
	check(false)
	notice.Status = couchmessage.Submitted
	proofs[notice.Message.ID] = notice
	check(true)
	after := slashOracleReceipt("after-command", couchmessage.Delivering)
	current = append(current, slashOracleInput(6, couchmessage.Envelope(after.Message)))
	proofs[after.Message.ID] = after
	check(false)
	after.Status = couchmessage.Submitted
	proofs[after.Message.ID] = after
	check(true)
	check(true)
	// Already observed envelopes must be revalidated against their typed proof.
	delete(proofs, notice.Message.ID)
	if _, err := oracle.Observe(current, proofs); err == nil {
		t.Fatal("missing previously observed proof accepted")
	}
}

func TestSlashJournalOracleRejectsUnknownAndDuplicateAdditions(t *testing.T) {
	hub, worker := slashOracleBindings()
	notice := slashOracleReceipt("notice-1", couchmessage.Submitted)
	envelope := couchmessage.Envelope(notice.Message)
	for _, tc := range []struct {
		name             string
		baseline, suffix []string
		proofs           map[string]couchmessage.Receipt
	}{
		{name: "ordinary-prose", suffix: []string{"/model", "ordinary prose"}},
		{name: "command-with-space", suffix: []string{"/model "}},
		{name: "unknown-system", suffix: []string{envelope, "/model"}},
		{name: "malformed-header", suffix: []string{"[Couch system; delivery ]\nnotice", "/model"}},
		{name: "duplicate-command", suffix: []string{"/model", "/model"}},
		{name: "duplicate-notice", suffix: []string{envelope, "/model", envelope}, proofs: map[string]couchmessage.Receipt{"notice-1": notice}},
		{name: "notice-replays-baseline", baseline: []string{envelope}, suffix: []string{"/model", envelope}, proofs: map[string]couchmessage.Receipt{"notice-1": notice}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var baseline []agentfixture.AcceptedInput
			for i, body := range tc.baseline {
				baseline = append(baseline, slashOracleInput(uint64(i+1), body))
			}
			current := append([]agentfixture.AcceptedInput(nil), baseline...)
			for _, body := range tc.suffix {
				current = append(current, slashOracleInput(uint64(len(current)+1), body))
			}
			if ready, err := newSlashJournalOracle(baseline, hub, worker).Observe(current, tc.proofs); err == nil || ready {
				t.Fatalf("unexpected addition accepted: %v %v", ready, err)
			}
		})
	}
}

func TestSlashJournalOraclePinsEveryStructFieldAndTruncation(t *testing.T) {
	hub, worker := slashOracleBindings()
	notice := slashOracleReceipt("notice-1", couchmessage.Submitted)
	baseline := []agentfixture.AcceptedInput{slashOracleInput(1, "startup")}
	full := append(append([]agentfixture.AcceptedInput(nil), baseline...), slashOracleInput(2, couchmessage.Envelope(notice.Message)), slashOracleInput(3, "/model"))
	typ := reflect.TypeOf(agentfixture.AcceptedInput{})
	for _, status := range []couchmessage.Status{couchmessage.Delivering, couchmessage.Submitted} {
		notice.Status = status
		proofs := map[string]couchmessage.Receipt{"notice-1": notice}
		for index := range full {
			for i := 0; i < typ.NumField(); i++ {
				field := typ.Field(i).Name
				t.Run(fmt.Sprintf("%s/%d/%s", status, index, field), func(t *testing.T) {
					oracle := newSlashJournalOracle(baseline, hub, worker)
					if _, err := oracle.Observe(full, proofs); err != nil {
						t.Fatal(err)
					}
					changed := append([]agentfixture.AcceptedInput(nil), full...)
					value := reflect.ValueOf(&changed[index]).Elem().FieldByName(field)
					switch value.Kind() {
					case reflect.String:
						value.SetString(value.String() + "-changed")
					case reflect.Int:
						value.SetInt(value.Int() + 1)
					case reflect.Uint64:
						value.SetUint(value.Uint() + 1)
					default:
						changed[index].At = changed[index].At.Add(time.Nanosecond)
					}
					if _, err := oracle.Observe(changed, proofs); err == nil {
						t.Fatal("pinned prefix field mutation accepted")
					}
				})
			}
		}
		for length := 0; length < len(full); length++ {
			t.Run(fmt.Sprintf("%s/truncate/%d", status, length), func(t *testing.T) {
				oracle := newSlashJournalOracle(baseline, hub, worker)
				if _, err := oracle.Observe(full, proofs); err != nil {
					t.Fatal(err)
				}
				if _, err := oracle.Observe(full[:length], proofs); err == nil {
					t.Fatal("observed prefix truncation accepted")
				}
			})
		}
	}
}

func TestSlashJournalOracleClonesInputsAndDoesNotPinErrors(t *testing.T) {
	hub, worker := slashOracleBindings()
	notice := slashOracleReceipt("notice-1", couchmessage.Queued)
	baseline := []agentfixture.AcceptedInput{slashOracleInput(1, "startup")}
	preserved := append([]agentfixture.AcceptedInput(nil), baseline...)
	oracle := newSlashJournalOracle(baseline, hub, worker)
	baseline[0].Body = "constructor alias mutation"
	if ready, err := oracle.Observe(preserved, nil); err != nil || ready {
		t.Fatalf("constructor retained baseline alias: %v %v", ready, err)
	}
	current := append(append([]agentfixture.AcceptedInput(nil), preserved...), slashOracleInput(2, couchmessage.Envelope(notice.Message)))
	proofs := map[string]couchmessage.Receipt{"notice-1": notice}
	if ready, err := oracle.Observe(current, proofs); err != nil || ready {
		t.Fatalf("pending notice: %v %v", ready, err)
	}
	preserved = append([]agentfixture.AcceptedInput(nil), current...)
	current[1].PID++
	current[0].Body = "observation alias mutation"
	invalid := append(append([]agentfixture.AcceptedInput(nil), preserved...), slashOracleInput(3, "untrusted prose"))
	if _, err := oracle.Observe(invalid, proofs); err == nil {
		t.Fatal("unknown input accepted")
	}
	notice.Status = couchmessage.Submitted
	proofs[notice.Message.ID] = notice
	valid := append(preserved, slashOracleInput(3, "/model"))
	if ready, err := oracle.Observe(valid, proofs); err != nil || !ready {
		t.Fatalf("retained caller alias or pinned failed observation: %v %v", ready, err)
	}
}

func TestSlashJournalOracleJournalBound(t *testing.T) {
	hub, worker := slashOracleBindings()
	baseline := make([]agentfixture.AcceptedInput, agentfixture.MaxRequests-1)
	for i := range baseline {
		baseline[i] = slashOracleInput(uint64(i+1), "startup")
	}
	current := append(append([]agentfixture.AcceptedInput(nil), baseline...), slashOracleInput(agentfixture.MaxRequests, "/model"))
	oracle := newSlashJournalOracle(baseline, hub, worker)
	if ready, err := oracle.Observe(current, nil); err != nil || !ready {
		t.Fatalf("bounded journal rejected: %v %v", ready, err)
	}
	current = append(current, slashOracleInput(agentfixture.MaxRequests+1, "/model"))
	if _, err := oracle.Observe(current, nil); err == nil {
		t.Fatal("oversized current journal accepted")
	}
	if _, err := newSlashJournalOracle(current, hub, worker).Observe(current, nil); err == nil {
		t.Fatal("oversized baseline accepted")
	}
}

func TestSlashJournalOracleCapturedDelayedRestartEnvelope(t *testing.T) {
	// This literal capture is the independent input; do not reconstruct it with
	// the current producer's restart-notice wording or its Envelope formatter.
	raw, err := os.ReadFile("testdata/slash-journal/delayed-restart-envelope.txt")
	if err != nil {
		t.Fatal(err)
	}
	captured := string(raw)
	const id = "a2965831-4006-4231-910e-52b5c2a34aab"
	const header = "[Couch system; delivery " + id + "]\n"
	body, ok := strings.CutPrefix(captured, header)
	if !ok {
		t.Fatal("captured delayed-restart envelope has unexpected first header")
	}
	hub, worker := slashOracleBindings()
	current := []agentfixture.AcceptedInput{slashOracleInput(1, "/model"), slashOracleInput(2, captured)}
	// The fake broker retains one immutable message while its receipt moves
	// through delivery and submission, independently of the observed journal.
	proof := couchmessage.Receipt{Message: couchmessage.Message{ID: id, Origin: couchmessage.OriginSystem, From: hub, To: worker, Body: body}, Status: couchmessage.Delivering}
	broker := map[string]couchmessage.Receipt{id: proof}
	oracle := newSlashJournalOracle(nil, hub, worker)
	if ready, err := oracle.Observe(current, broker); err != nil || ready {
		t.Fatalf("captured notice delivering: ready=%v err=%v", ready, err)
	}
	proof.Status = couchmessage.Submitted
	broker[id] = proof
	if ready, err := oracle.Observe(current, broker); err != nil || !ready {
		t.Fatalf("captured notice submitted: ready=%v err=%v", ready, err)
	}
	if ready, err := newSlashJournalOracle(nil, hub, worker).Observe(current, nil); err == nil || ready {
		t.Fatal("literal display header trusted without broker receipt")
	}
	proof.Message.Origin = couchmessage.OriginPeer
	broker[id] = proof
	if ready, err := newSlashJournalOracle(nil, hub, worker).Observe(current, broker); err == nil || ready {
		t.Fatal("literal system header trusted with peer origin")
	}
}

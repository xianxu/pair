package zellijquery

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/zellijreply"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

func TestLookupMissRequiresAnExplicitSession(t *testing.T) {
	args := []string{"action", "list-clients"}
	if RetryableSessionLookupMiss(args, nil, []byte("There is no active session!\n"), 1, true) {
		t.Fatal("implicit session read borrowed a socket identity")
	}
}

func FuzzReadKind(f *testing.F) {
	// Enumerate the accepted grammar independently, including every subset and
	// ordering of unique pane flags. JSON keys avoid token-boundary collisions.
	key := func(args []string) string { raw, _ := json.Marshal(args); return string(raw) }
	allowed := map[string]ReplyKind{
		key([]string{"list-sessions", "--no-formatting"}): SessionsReply,
	}
	add := func(args []string, kind ReplyKind) {
		allowed[key(args)] = kind
		allowed[key(append([]string{"--session", "SESSION"}, args...))] = kind
	}
	add([]string{"action", "list-clients"}, ClientsReply)
	add([]string{"action", "current-tab-info", "--json"}, TabReply)
	flags := []string{"--json", "--command", "--state", "--geometry", "--tab", "--all"}
	var enumerate func([]string, uint8)
	enumerate = func(chosen []string, used uint8) {
		if used&1 != 0 {
			add(append([]string{"action", "list-panes"}, chosen...), PanesReply)
		}
		for i, flag := range flags {
			if used&(1<<i) == 0 {
				enumerate(append(append([]string(nil), chosen...), flag), used|1<<i)
			}
		}
	}
	enumerate(nil, 0)
	f.Add("action|list-panes|--json|--geometry")
	f.Add("--session|test|action|list-clients")
	f.Add("action|close-pane|--json")
	f.Add("action|list-panes|--json|--json")
	f.Add("action|list-panes|--json|--future")
	f.Fuzz(func(t *testing.T, raw string) {
		args := strings.Split(raw, "|")
		kind := ReadKind(args)
		if len(args) >= 2 && args[0] == "--session" && args[1] != "" {
			args[1] = "SESSION"
		}
		if want := allowed[key(args)]; kind != want {
			t.Fatalf("argv=%q kind=%d want=%d", args, kind, want)
		}
	})
}

func FuzzCrossedReply(f *testing.F) {
	f.Add(uint8(PanesReply), []byte(zellijtest.Reply(f, "panes", []zellijtest.Pane{}).Stdout))
	f.Add(uint8(PanesReply), []byte(zellijtest.Reply(f, "clients", []zellijtest.Client{}).Stdout))
	f.Add(uint8(ClientsReply), []byte(zellijtest.Reply(f, "tab", zellijtest.DefaultTab()).Stdout))
	f.Add(uint8(255), []byte("[]"))
	f.Add(uint8(ClientsReply), []byte("[{"))
	f.Add(uint8(PanesReply), []byte{})
	f.Fuzz(func(t *testing.T, kind uint8, raw []byte) {
		got := CrossedReply(ReplyKind(kind), raw)
		_, clientErr := zellijreply.ParseClients(raw)
		_, paneErr := zellijreply.ParsePanes(raw, zellijreply.PaneRequirements{})
		_, tabErr := zellijreply.ParseTab(raw)
		valid := map[ReplyKind]bool{SessionsReply: ValidateSessionListing(raw, false) == nil,
			ClientsReply: clientErr == nil, PanesReply: paneErr == nil, TabReply: tabErr == nil}
		want := false
		if kind > 0 && kind <= uint8(TabReply) && len(bytes.TrimSpace(raw)) > 0 && !valid[ReplyKind(kind)] {
			for _, complete := range valid {
				want = want || complete
			}
		}
		if got != want {
			t.Fatalf("kind=%d raw=%q crossed=%v want=%v", kind, raw, got, want)
		}
	})
}

func FuzzSessionReplyValidation(f *testing.F) {
	f.Add([]byte("pair-a [Created 0s ago]\n"))
	f.Add([]byte("\x1b[32;1mtest\x1b[m [Created \x1b[35;1m0s\x1b[m ago] \n"))
	f.Fuzz(func(t *testing.T, raw []byte) {
		states, err := SessionListingStates(raw)
		if err == nil && ValidateSessionListing(raw, false) != nil {
			t.Fatal("derived identity from an invalid listing")
		}
		if err != nil && states != nil {
			t.Fatal("invalid listing retained authority")
		}
		_, _ = ParseColoredSessionListing(raw)
	})
}

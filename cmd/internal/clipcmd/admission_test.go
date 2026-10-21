package clipcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

type admissionRuntime struct {
	OSRuntime
	notices []string
	resets  []string
}

func (r *admissionRuntime) ClipboardPaste() (string, bool) { return "selected body", true }
func (r *admissionRuntime) NotifyRefusal(s string) error {
	r.notices = append(r.notices, s)
	return nil
}
func (r *admissionRuntime) ResetPaneColorAfter(id string, _ time.Duration) error {
	r.resets = append(r.resets, id)
	return nil
}

func clipboardFixture(t *testing.T) (*zellijtest.Fixture, *admissionRuntime, ClipboardToPaneOptions) {
	t.Helper()
	f := zellijtest.New(t)
	for _, entry := range f.Env() {
		k, v, _ := strings.Cut(entry, "=")
		t.Setenv(k, v)
	}
	agent := zellijtest.DefaultPane(0, "pair wrap claude")
	agent["terminal_command"] = "pair wrap claude"
	draft := zellijtest.DefaultPane(1, "nvim -u /pair/nvim/init.lua /data/draft-t.md")
	draft["terminal_command"] = "nvim -u /pair/nvim/init.lua /data/draft-t.md"
	draft["is_focused"] = false
	f.SetModel(zellijtest.Model{Sessions: map[string]zellijtest.Session{"test": {Panes: []zellijtest.Pane{agent, draft}, Clients: []zellijtest.Client{{ID: 1, PaneID: "terminal_0", RunningCommand: "pair wrap claude"}}, Tab: zellijtest.DefaultTab()}}})
	dataDir := t.TempDir()
	t.Setenv("PAIR_DATA_DIR", dataDir)
	t.Setenv("PAIR_TAG", "t")
	return f, &admissionRuntime{}, ClipboardToPaneOptions{DataDir: dataDir, Tag: "t"}
}
func TestClipboardAdmissionRefusesBeforeStaging(t *testing.T) {
	for _, kind := range []string{"wrong-kind", "empty", "missing-draft", "ambiguous-draft", "changed-draft"} {
		t.Run(kind, func(t *testing.T) {
			f, rt, opts := clipboardFixture(t)
			model := f.Model()
			session := model.Sessions["test"]
			switch kind {
			case "wrong-kind":
				raw := `{"active_swap_layout_name":null,"is_swap_layout_dirty":false}`
				f.SetFaults(zellijtest.Fault{Ordinal: 1, Reply: &raw})
			case "empty":
				raw := ""
				f.SetFaults(zellijtest.Fault{Ordinal: 1, Reply: &raw})
			case "missing-draft":
				session.Panes = session.Panes[:1]
				model.Sessions["test"] = session
				f.SetModel(model)
			case "ambiguous-draft":
				duplicate := zellijtest.DefaultPane(2, "nvim -u /pair/nvim/init.lua /data/draft-t.md")
				duplicate["terminal_command"] = "nvim -u /pair/nvim/init.lua /data/draft-t.md"
				session.Panes = append(session.Panes, duplicate)
				model.Sessions["test"] = session
				f.SetModel(model)
			case "changed-draft":
				session.Panes[1]["id"] = 4
				model.Sessions["test"] = session
				f.SetFaults(zellijtest.Fault{Ordinal: 2, Model: &model})
			}
			code := RunClipboardToPane(opts, rt, io.Discard)
			if code == 0 {
				t.Errorf("uncertain %s admitted", kind)
			}
			if _, err := os.Stat(filepath.Join(opts.DataDir, "quote-t")); !os.IsNotExist(err) {
				t.Errorf("quote staged before admission: %v", err)
			}
			if len(f.Effects()) != 0 {
				t.Errorf("effects=%v", f.Effects())
			}
			if len(rt.notices) != 1 {
				t.Errorf("refusal not presented: %v", rt.notices)
			}
		})
	}
}

func TestAutoQuoteAdmissionUsesCombinedSourceContext(t *testing.T) {
	for _, scenario := range []string{"healthy", "multiple-tabs", "unanimous-clients", "disagreeing-clients", "missing-client", "missing-source", "missing-draft", "other-tab-draft", "ambiguous-draft", "wrong-clients", "wrong-panes", "changed-source", "changed-draft", "plain-nvim", "right-terminal", "unregistered-terminal", "registered-split"} {
		t.Run(scenario, func(t *testing.T) {
			f, rt, opts := clipboardFixture(t)
			model := f.Model()
			s := model.Sessions["test"]
			switch scenario {
			case "multiple-tabs":
				p := zellijtest.DefaultPane(9, "nvim -u /pair/nvim/init.lua /other/draft.md")
				p["terminal_command"] = "nvim -u /pair/nvim/init.lua /other/draft.md"
				p["tab_id"] = 2
				p["tab_position"] = 1
				s.Panes = append(s.Panes, p)
			case "unanimous-clients":
				s.Clients = append(s.Clients, zellijtest.Client{ID: 2, PaneID: "terminal_0", RunningCommand: "pair wrap claude"})
			case "disagreeing-clients":
				s.Clients = append(s.Clients, zellijtest.Client{ID: 2, PaneID: "terminal_1", RunningCommand: "nvim"})
			case "missing-client":
				s.Clients = nil
			case "missing-source":
				s.Clients[0].PaneID = "terminal_8"
			case "missing-draft":
				s.Panes = s.Panes[:1]
			case "other-tab-draft":
				s.Panes[1]["tab_id"] = 2
				s.Panes[1]["tab_position"] = 1
			case "ambiguous-draft":
				p := zellijtest.DefaultPane(3, "nvim -u /pair/nvim/init.lua /data/draft-t.md")
				p["terminal_command"] = "nvim -u /pair/nvim/init.lua /data/draft-t.md"
				s.Panes = append(s.Panes, p)
			case "plain-nvim":
				s.Panes[0]["terminal_command"] = "nvim /tmp/other.md"
			case "right-terminal", "unregistered-terminal":
				s.Panes[0]["terminal_command"] = "pair term"
			case "registered-split":
				s.Panes[0]["terminal_command"] = nil
				t.Setenv("PAIR_TERMINAL_PANES_PATH", "")
			}
			model.Sessions["test"] = s
			f.SetModel(model)
			switch scenario {
			case "wrong-clients":
				raw := `[]`
				f.SetFaults(zellijtest.Fault{Ordinal: 1, Reply: &raw})
			case "wrong-panes":
				raw := "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n"
				f.SetFaults(zellijtest.Fault{Ordinal: 2, Reply: &raw})
			case "changed-source":
				s.Clients[0].PaneID = "terminal_1"
				model.Sessions["test"] = s
				f.SetFaults(zellijtest.Fault{Ordinal: 3, Model: &model})
			case "changed-draft":
				s.Panes[1]["id"] = 4
				model.Sessions["test"] = s
				f.SetFaults(zellijtest.Fault{Ordinal: 4, Model: &model})
			}
			var runtime Runtime = rt
			if scenario == "registered-split" || scenario == "right-terminal" {
				runtime = &registeredAdmissionRuntime{admissionRuntime: rt}
			}
			code := RunCopyOnSelectOrchestrate(CopyOnSelectOptions{Clipboard: opts}, runtime, io.Discard)
			healthy := scenario == "healthy" || scenario == "multiple-tabs" || scenario == "unanimous-clients"
			skip := scenario == "plain-nvim" || scenario == "right-terminal" || scenario == "registered-split"
			if healthy {
				if code != 0 {
					t.Fatalf("healthy refused code=%d notices=%v", code, rt.notices)
				}
				if body := f.Model().Sessions["test"].Panes[1]["input"]; body != string([]byte{31}) {
					t.Fatalf("paste input=%v effects=%v", body, f.Effects())
				}
				if len(f.Effects()) != 3 || f.Effects()[0].Action != "set-pane-color" || f.Effects()[0].Target != "terminal_0" || f.Effects()[1].Target != "terminal_1" || f.Effects()[2].Target != "terminal_1" {
					t.Fatalf("effects=%v", f.Effects())
				}
			} else {
				if skip && code != 0 || !skip && code == 0 {
					t.Fatalf("code=%d skip=%v notices=%v", code, skip, rt.notices)
				}
				if len(f.Effects()) != 0 {
					t.Fatalf("refused/skip effects=%v", f.Effects())
				}
				if _, err := os.Stat(filepath.Join(opts.DataDir, "quote-t")); !os.IsNotExist(err) {
					t.Fatalf("quote staged: %v", err)
				}
				if !skip && len(rt.notices) != 1 {
					t.Fatalf("missing notice=%v", rt.notices)
				}
			}
			want := 4
			switch scenario {
			case "wrong-clients", "disagreeing-clients", "missing-client":
				want = 1
			case "wrong-panes", "missing-source", "missing-draft", "other-tab-draft", "ambiguous-draft", "unregistered-terminal":
				want = 2
			}
			observations := 0
			for _, q := range f.Queries() {
				if len(q.Args) > 1 && (q.Args[1] == "list-clients" || q.Args[1] == "list-panes") {
					observations++
				}
			}
			if observations != want {
				t.Fatalf("observation calls=%d want=%d queries=%v", observations, want, f.Queries())
			}
		})
	}
}

type registeredAdmissionRuntime struct{ *admissionRuntime }

func (*registeredAdmissionRuntime) TerminalPaneIDs() ([]string, error) { return []string{"0"}, nil }

func TestClipboardNativeFocusFailureStopsTrigger(t *testing.T) {
	f, rt, opts := clipboardFixture(t)
	f.SetFaults(zellijtest.Fault{Ordinal: 3, Error: "native focus failed"})
	if code := RunClipboardToPane(opts, rt, io.Discard); code == 0 {
		t.Fatal("focus failure accepted")
	}
	if len(f.Queries()) != 3 || len(f.Effects()) != 0 {
		t.Fatalf("focus failure replayed/triggered: queries=%v effects=%v", f.Queries(), f.Effects())
	}
}

type contextAdmissionRuntime struct {
	*admissionRuntime
	contexts    []context.Context
	cancel      context.CancelFunc
	cancelAfter int
}

func (r *contextAdmissionRuntime) ListClientsContext(ctx context.Context) ([]byte, error) {
	r.contexts = append(r.contexts, ctx)
	raw, err := r.OSRuntime.ListClientsContext(ctx)
	if len(r.contexts) == r.cancelAfter {
		r.cancel()
	}
	return raw, err
}
func (r *contextAdmissionRuntime) ListPanesContext(ctx context.Context) ([]byte, error) {
	r.contexts = append(r.contexts, ctx)
	raw, err := r.OSRuntime.ListPanesContext(ctx)
	if len(r.contexts) == r.cancelAfter {
		r.cancel()
	}
	return raw, err
}
func TestClipboardAdmissionSharesCallerDeadline(t *testing.T) {
	for _, cancelAfter := range []int{0, 2} {
		t.Run(fmt.Sprint(cancelAfter), func(t *testing.T) {
			f, base, _ := clipboardFixture(t)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			rt := &contextAdmissionRuntime{admissionRuntime: base, cancel: cancel, cancelAfter: cancelAfter}
			_, err := admitAutoQuote(ctx, rt)
			if cancelAfter == 0 && err != nil || cancelAfter != 0 && !errors.Is(err, context.Canceled) {
				t.Fatalf("admission err=%v", err)
			}
			want := 4
			if cancelAfter != 0 {
				want = 2
			}
			if len(rt.contexts) != want {
				t.Fatalf("reads=%d want=%d", len(rt.contexts), want)
			}
			for _, got := range rt.contexts {
				if got != ctx {
					t.Fatal("read replaced operation context")
				}
			}
			if len(f.Effects()) != 0 {
				t.Fatalf("preflight effects=%v", f.Effects())
			}
		})
	}
}

func TestStandaloneClipboardPinsInvokingTab(t *testing.T) {
	for _, scenario := range []string{"other-tab-draft", "missing-caller", "missing-identity", "changed-caller-tab", "contradictory-position"} {
		t.Run(scenario, func(t *testing.T) {
			f, rt, opts := clipboardFixture(t)
			m := f.Model()
			s := m.Sessions["test"]
			healthy := scenario == "other-tab-draft"
			switch scenario {
			case "other-tab-draft":
				p := zellijtest.DefaultPane(7, "nvim -u /pair/nvim/init.lua /other/draft.md")
				p["terminal_command"] = "nvim -u /pair/nvim/init.lua /other/draft.md"
				p["tab_id"] = 2
				p["tab_position"] = 1
				s.Panes = append(s.Panes, p)
			case "missing-caller":
				t.Setenv("ZELLIJ_PANE_ID", "8")
			case "missing-identity":
				t.Setenv("ZELLIJ_PANE_ID", "")
			case "contradictory-position":
				s.Panes[1]["tab_position"] = 7
			}
			m.Sessions["test"] = s
			f.SetModel(m)
			if scenario == "changed-caller-tab" {
				s.Panes[0]["tab_id"] = 2
				s.Panes[0]["tab_position"] = 1
				m.Sessions["test"] = s
				f.SetFaults(zellijtest.Fault{Ordinal: 2, Model: &m})
			}
			code := RunClipboardToPane(opts, rt, io.Discard)
			if healthy {
				if code != 0 || len(f.Effects()) != 2 || f.Effects()[1].Target != "terminal_1" {
					t.Fatalf("healthy invoking tab: code=%d effects=%v", code, f.Effects())
				}
			} else {
				if code == 0 || len(f.Effects()) != 0 {
					t.Fatalf("invalid caller admitted: code=%d effects=%v", code, f.Effects())
				}
				if _, err := os.Stat(filepath.Join(opts.DataDir, "quote-t")); !os.IsNotExist(err) {
					t.Fatalf("invalid caller staged=%v", err)
				}
			}
		})
	}
}

func TestFlashAdmissionKeepsExplicitTargetAuthority(t *testing.T) {
	for _, scenario := range []string{"explicit", "observed", "wrong-kind"} {
		t.Run(scenario, func(t *testing.T) {
			f, rt, _ := clipboardFixture(t)
			opts := FlashPaneOptions{}
			if scenario == "explicit" {
				opts.PaneID = "1"
			}
			if scenario == "wrong-kind" {
				raw := `[]`
				f.SetFaults(zellijtest.Fault{Ordinal: 1, Reply: &raw})
			}
			code := RunFlashPane(opts, rt, io.Discard)
			if scenario == "wrong-kind" {
				if code == 0 || len(f.Effects()) != 0 || len(f.Queries()) != 1 {
					t.Fatalf("invalid source effects=%v queries=%v", f.Effects(), f.Queries())
				}
				return
			}
			target := "terminal_0"
			observations := 2
			if scenario == "explicit" {
				target = "terminal_1"
				observations = 0
			}
			if code != 0 || len(f.Effects()) != 1 || f.Effects()[0].Target != target || len(f.Queries()) != observations+1 {
				t.Fatalf("flash code=%d effects=%v queries=%v", code, f.Effects(), f.Queries())
			}
		})
	}
}

func TestClipboardFocusUsesExactDesiredStateSuccess(t *testing.T) {
	for _, message := range []string{"Pane Terminal(1) is already focused", "Pane Terminal(2) is already focused", "other failure"} {
		t.Run(message, func(t *testing.T) {
			root := t.TempDir()
			calls := filepath.Join(root, "calls")
			t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("PAIR_TEST_FOCUS_CALLS", calls)
			t.Setenv("PAIR_TEST_FOCUS_ERROR", message)
			script := "#!/bin/sh\nprintf 'call\\n' >> \"$PAIR_TEST_FOCUS_CALLS\"\nprintf '%s\\n' \"$PAIR_TEST_FOCUS_ERROR\" >&2\nexit 2\n"
			if err := os.WriteFile(filepath.Join(root, "zellij"), []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			err := (OSRuntime{}).FocusPaneContext(t.Context(), "1")
			if (err == nil) != (message == "Pane Terminal(1) is already focused") {
				t.Fatalf("focus err=%v", err)
			}
			raw, readErr := os.ReadFile(calls)
			if readErr != nil || string(raw) != "call\n" {
				t.Fatalf("focus retried: %q,%v", raw, readErr)
			}
		})
	}
}

func TestAutoQuoteValidatesQuoteArtifactBeforeFlash(t *testing.T) {
	f, rt, opts := clipboardFixture(t)
	opts.Tag = "../invalid"
	if code := RunCopyOnSelectOrchestrate(CopyOnSelectOptions{Clipboard: opts}, rt, io.Discard); code == 0 {
		t.Fatal("invalid quote artifact accepted")
	}
	if len(f.Effects()) != 0 || len(rt.resets) != 0 {
		t.Fatalf("artifact failure after effects=%v resets=%v", f.Effects(), rt.resets)
	}
}

type noticeFailureRuntime struct {
	*admissionRuntime
	logs []string
}

func (r *noticeFailureRuntime) NotifyRefusal(string) error { return errors.New("broker unavailable") }
func (r *noticeFailureRuntime) Log(s string)               { r.logs = append(r.logs, s) }
func TestClipboardRefusalRetainsPresentationFailure(t *testing.T) {
	f, base, opts := clipboardFixture(t)
	raw := `{}`
	f.SetFaults(zellijtest.Fault{Ordinal: 1, Reply: &raw})
	rt := &noticeFailureRuntime{admissionRuntime: base}
	var stderr bytes.Buffer
	if code := RunClipboardToPane(opts, rt, &stderr); code == 0 {
		t.Fatal("invalid observation admitted")
	}
	if !strings.Contains(stderr.String(), "refusal notification failed: broker unavailable") || !strings.Contains(strings.Join(rt.logs, "\n"), "refusal notification failed: broker unavailable") {
		t.Fatalf("presentation failure lost: stderr=%s log=%v", &stderr, rt.logs)
	}
	if strings.Contains(stderr.String(), "selected body") {
		t.Fatal("clipboard text leaked into refusal")
	}
	if len(f.Effects()) != 0 {
		t.Fatalf("refusal effects=%v", f.Effects())
	}
}

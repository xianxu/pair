package couchcmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessionsync"
)

// sessionSyncArgs is one `couch --session-sync` invocation (pair#437).
type sessionSyncArgs struct {
	verb, file, section, bodyFile, ifHash string
	all, json                             bool
	accepts                               sessionsync.Accepts
	fold                                  []string
}

const sessionSyncUsage = `couch --session-sync diff FILE
couch --session-sync status [--json] FILE
couch --session-sync read --section HEADING FILE [--json]
couch --session-sync read --all FILE [--json]
couch --session-sync write --section HEADING BODYFILE FILE --if-hash H [--accept-gap] [--fold ID[,ID]]
couch --session-sync write --all BODYFILE FILE --if-hash H [--accept-gap] [--fold ID[,ID]]`

// parseSessionSyncCLI reads the closed forms above. Values (a heading, a
// path) are taken verbatim even when they start with "--".
func parseSessionSyncCLI(args []string) (cliInvocation, error) {
	usage := func(format string, v ...any) (cliInvocation, error) {
		return cliInvocation{}, fmt.Errorf("%s\nusage:\n%s", fmt.Sprintf(format, v...), sessionSyncUsage)
	}
	if len(args) < 2 {
		return usage("--session-sync needs a verb")
	}
	a := &sessionSyncArgs{verb: args[1]}
	allowed := map[string][]string{
		"diff":   {},
		"status": {"--json"},
		"read":   {"--section", "--all", "--json"},
		"write":  {"--section", "--all", "--if-hash", "--accept-gap", "--fold"},
		// Retired by pair#439 (no agent-held lock); accepted as a no-op
		// with its old flags for one release so older skills don't break.
		"release": {"--accept-edits", "--accept-gap", "--fold"},
	}
	flags, ok := allowed[a.verb]
	if !ok {
		return usage("unknown --session-sync verb %q", a.verb)
	}
	var positional []string
	seen := map[string]bool{}
	rest := args[2:]
	for i := 0; i < len(rest); i++ {
		arg := rest[i]
		if !strings.HasPrefix(arg, "--") {
			positional = append(positional, arg)
			continue
		}
		known := false
		for _, f := range flags {
			known = known || f == arg
		}
		if !known || seen[arg] {
			return usage("--session-sync %s does not take %s here", a.verb, arg)
		}
		seen[arg] = true
		switch arg {
		case "--section", "--fold", "--if-hash":
			if i+1 >= len(rest) || rest[i+1] == "" {
				return usage("%s needs a value", arg)
			}
			i++
			if arg == "--section" {
				a.section = rest[i]
			} else if arg == "--if-hash" {
				a.ifHash = rest[i]
			} else {
				for _, id := range strings.Split(rest[i], ",") {
					if id = strings.TrimSpace(id); id != "" {
						a.fold = append(a.fold, id)
					}
				}
			}
		case "--all":
			a.all = true
		case "--accept-gap":
			a.accepts.Gap = true
		case "--json":
			a.json = true
		}
	}
	want := 1
	if (a.verb == "write" || a.verb == "read") && seen["--section"] == a.all {
		return usage("%s takes exactly one of --section HEADING or --all", a.verb)
	}
	if a.verb == "write" {
		if a.ifHash == "" {
			return usage("write needs --if-hash H, the hash `read` printed (an unconditional write could overwrite an edit you never saw)")
		}
		want = 2
	}
	if len(positional) != want {
		return usage("--session-sync %s takes %d path argument(s), got %d", a.verb, want, len(positional))
	}
	a.file = positional[want-1]
	if want == 2 {
		a.bodyFile = positional[0]
	}
	return cliInvocation{kind: cliSessionSync, sessionSync: a}, nil
}

// runSessionSyncCLI runs in-process: it reads the store's ledger directly
// and constructs no supervisor, so it answers whether or not Couch runs
// (status alone refuses when no Couch has recorded deliveries).
func runSessionSyncCLI(a *sessionSyncArgs, rt Runtime, stdout, stderr io.Writer) int {
	fail := func(err error) int {
		code, exit := "error", 1
		var r *sessionsync.Refusal
		if errors.As(err, &r) {
			code, exit = r.Code, r.Exit
		}
		if a.json {
			_ = json.NewEncoder(stdout).Encode(sessionSyncResponse{Code: code, Error: err.Error(), Hash: refusalHash(err)})
		} else {
			fmt.Fprintln(stderr, "couch: refused:", err)
			if h := refusalHash(err); h != "" {
				fmt.Fprintln(stdout, "hash:", h)
			}
		}
		return exit
	}
	if a.verb == "release" {
		fmt.Fprintln(stderr, "couch: release is retired and did nothing: the agent holds no lock (pair#439); fold events with `write --fold ID`")
		return 0
	}
	file, err := filepath.Abs(a.file)
	if err != nil {
		return fail(err)
	}
	namespace, err := rt.ResolveNamespace()
	if err != nil {
		return fail(err)
	}
	s := sessionsync.New(namespace.Dir())
	switch a.verb {
	case "status":
		st, err := s.Status(file)
		if err != nil {
			return fail(err)
		}
		if a.json {
			if err := json.NewEncoder(stdout).Encode(sessionSyncResponse{Code: "ok", Status: &st}); err != nil {
				return fail(err)
			}
			return 0
		}
		fmt.Fprint(stdout, formatSessionSyncStatus(st))
	case "diff":
		holder, diff, err := s.DiffText(file)
		if err != nil {
			return fail(err)
		}
		fmt.Fprintf(stdout, "lock: %s\n%s", lockText(holder), diff)
	case "read":
		r, err := s.Read(file, a.section)
		if err != nil {
			return fail(err)
		}
		if a.json {
			if err := json.NewEncoder(stdout).Encode(sessionSyncResponse{Code: "ok", Read: &r}); err != nil {
				return fail(err)
			}
			return 0
		}
		fmt.Fprintf(stdout, "hash: %s\n%s", r.Hash, r.Body)
	case "write":
		raw, err := os.ReadFile(a.bodyFile)
		if err != nil {
			return fail(fmt.Errorf("reading the body: %w", err))
		}
		res, err := s.Write(sessionsync.WriteRequest{File: file, Section: a.section, Body: string(raw),
			IfHash: a.ifHash, Accepts: a.accepts, Fold: a.fold})
		if err != nil {
			return fail(err)
		}
		if a.section != "" {
			fmt.Fprintf(stdout, "wrote ## %s\n", a.section)
		} else {
			fmt.Fprintln(stdout, "rewrote the file")
		}
		fmt.Fprintln(stdout, "hash:", res.Hash)
		reportFolds(stdout, stderr, res)
	}
	return 0
}

// sessionSyncResponse is the Couch JSON envelope (Code, Error, payload),
// agreed with parley.nvim#314.
type sessionSyncResponse struct {
	Code   string
	Error  string
	Hash   string `json:",omitempty"` // the current hash, on a stale refusal
	Status *sessionsync.Status
	Read   *sessionsync.ReadResult `json:",omitempty"`
}

func refusalHash(err error) string {
	var r *sessionsync.Refusal
	if errors.As(err, &r) {
		return r.Hash
	}
	return ""
}

func lockText(h sessionsync.Holder) string {
	if h == sessionsync.HolderNone {
		return "none"
	}
	return string(h)
}

func reportFolds(stdout, stderr io.Writer, res sessionsync.Result) {
	if len(res.Folded) > 0 {
		fmt.Fprintf(stdout, "folded %s\n", strings.Join(res.Folded, ","))
	}
	if res.Warning != "" {
		fmt.Fprintln(stderr, "couch: warning:", res.Warning)
	}
}

func formatSessionSyncStatus(st sessionsync.Status) string {
	var b strings.Builder
	last := "never"
	if st.LastWrite != nil {
		last = st.LastWrite.Local().Format("2006-01-02 15:04:05")
	}
	fmt.Fprintf(&b, "lock: %s\nowner: %s\nlast write: %s\nowed: %d\n", lockText(st.Holder), st.Owner, last, len(st.Owed))
	for _, e := range st.Owed {
		note := ""
		if e.Uncertain {
			note = " (delivery uncertain)"
		}
		fmt.Fprintf(&b, "  %s  %-8s %s from %s%s\n", e.At.Local().Format(time.DateTime), e.Kind, e.ID, e.From, note)
	}
	if len(st.Owed) > 0 {
		fmt.Fprintln(&b, "fold each with `write --fold ID` once the file reflects it")
	}
	return b.String()
}

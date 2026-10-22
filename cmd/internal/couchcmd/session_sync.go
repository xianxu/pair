package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/sessionsync"
)

// sessionSyncArgs is one `couch --session-sync` invocation (pair#437).
type sessionSyncArgs struct {
	verb, file, section, bodyFile, ifHash string
	owner, oldFile                        string
	id, turn                              string
	all, frontmatter, json                bool
	accepts                               sessionsync.Accepts
	fold                                  []string
}

func (a *sessionSyncArgs) operatorLockCommand() bool {
	return a.verb == "operator-acquire" || a.verb == "operator-release"
}

// Preparation and namespace resolution can fail before the session-sync store
// exists. Both boundaries use the same machine-readable operator outcome.
func sessionSyncUnavailable(stdout io.Writer, err error) int {
	return operatorSessionSyncFailure(stdout, &sessionsync.Refusal{Code: "session-sync-unavailable", Exit: sessionsync.ExitUnavailable, Reason: err.Error()})
}

func operatorSessionSyncFailure(stdout io.Writer, err error) int {
	code, exit := "error", 1
	var refusal *sessionsync.Refusal
	if errors.As(err, &refusal) {
		code, exit = refusal.Code, refusal.Exit
	}
	_ = json.NewEncoder(stdout).Encode(sessionSyncResponse{Code: code, Error: err.Error()})
	return exit
}

const sessionSyncUsage = `couch --session-sync register FILE --owner SLOT [--json]
couch --session-sync lookup FILE [--json]
couch --session-sync list [--json]
couch --session-sync forget FILE [--json]
couch --session-sync rename OLD NEW [--json]
couch --session-sync diff FILE
couch --session-sync status [--json] FILE
couch --session-sync read --section HEADING FILE [--json]
couch --session-sync read --all FILE [--json]
couch --session-sync read --frontmatter FILE [--json]
couch --session-sync write --section HEADING BODYFILE FILE --if-hash H [--accept-gap] [--fold ID[,ID]]
couch --session-sync write --all BODYFILE FILE --if-hash H [--accept-gap] [--fold ID[,ID]]
couch --session-sync write --frontmatter BODYFILE FILE --if-hash H [--accept-gap] [--fold ID[,ID]]
couch --session-sync apply EDITS.json FILE --if-hash H [--accept-gap] [--fold ID[,ID]] [--json]
couch --session-sync round FILE --json
couch --session-sync operator-acquire FILE --id ID --turn TOKEN --json
couch --session-sync operator-release FILE --id ID --turn TOKEN --json`

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
		"register":         {"--owner", "--json"},
		"lookup":           {"--json"},
		"list":             {"--json"},
		"forget":           {"--json"},
		"rename":           {"--json"},
		"diff":             {},
		"status":           {"--json"},
		"read":             {"--section", "--all", "--frontmatter", "--json"},
		"write":            {"--section", "--all", "--frontmatter", "--if-hash", "--accept-gap", "--fold"},
		"apply":            {"--if-hash", "--accept-gap", "--fold", "--json"},
		"round":            {"--json"},
		"operator-acquire": {"--id", "--turn", "--json"},
		"operator-release": {"--id", "--turn", "--json"},
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
		case "--section", "--fold", "--if-hash", "--owner", "--id", "--turn":
			if i+1 >= len(rest) || rest[i+1] == "" {
				return usage("%s needs a value", arg)
			}
			i++
			if arg == "--owner" {
				a.owner = rest[i]
			} else if arg == "--id" {
				a.id = rest[i]
			} else if arg == "--turn" {
				a.turn = rest[i]
			} else if arg == "--section" {
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
		case "--frontmatter":
			a.frontmatter = true
		case "--accept-gap":
			a.accepts.Gap = true
		case "--json":
			a.json = true
		}
	}
	want := 1
	if a.verb == "list" {
		want = 0
	}
	if a.verb == "rename" {
		want = 2
	}
	if a.verb == "register" && a.owner == "" {
		return usage("register requires --owner SLOT")
	}
	if a.verb == "write" || a.verb == "read" {
		scopes := 0
		for _, flag := range []string{"--section", "--all", "--frontmatter"} {
			if seen[flag] {
				scopes++
			}
		}
		if scopes != 1 {
			return usage("%s takes exactly one of --section HEADING, --all or --frontmatter", a.verb)
		}
	}
	if a.verb == "apply" {
		if a.ifHash == "" {
			return usage("apply needs --if-hash H, the whole-file hash `read --all` printed")
		}
		want = 2
	}
	if a.verb == "round" && !a.json {
		return usage("round answers only as JSON: pass --json")
	}
	if a.operatorLockCommand() {
		if a.id == "" || a.turn == "" || !a.json {
			return usage("%s requires --id ID --turn TOKEN --json", a.verb)
		}
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
	if want > 0 {
		a.file = positional[want-1]
	}
	if want == 2 {
		if a.verb == "rename" {
			a.oldFile = positional[0]
		} else {
			a.bodyFile = positional[0]
		}
	}
	return cliInvocation{kind: cliSessionSync, sessionSync: a}, nil
}

// runSessionSyncCLI runs in-process: it reads the store's ledger directly
// and constructs no supervisor, so it answers whether or not Couch runs
// (status alone refuses when no Couch has recorded deliveries).
func runSessionSyncCLI(a *sessionSyncArgs, rt Runtime, stdout, stderr io.Writer) int {
	fail := func(err error) int {
		if a.operatorLockCommand() {
			return operatorSessionSyncFailure(stdout, err)
		}
		code, exit := "error", 1
		var r *sessionsync.Refusal
		if errors.As(err, &r) {
			code, exit = r.Code, r.Exit
		}
		var invalid []sessionsync.BadEdit
		if r != nil {
			invalid = r.Invalid
		}
		switch {
		case a.json && a.verb == "round":
			_ = json.NewEncoder(stdout).Encode(roundResponse{Code: code, Error: err.Error()})
		case a.json:
			_ = json.NewEncoder(stdout).Encode(sessionSyncResponse{Code: code, Error: err.Error(), Hash: refusalHash(err), Invalid: invalid})
		default:
			fmt.Fprintln(stderr, "couch: refused:", err)
			for _, b := range invalid {
				fmt.Fprintf(stderr, "  record %d: %s\n", b.Index, b.Reason)
			}
			if h := refusalHash(err); h != "" {
				fmt.Fprintln(stdout, "hash:", h)
			}
		}
		return exit
	}
	if a.verb == "release" {
		// A fold the caller asked for did not happen; say so with a failure.
		if len(a.fold) > 0 {
			return fail(fmt.Errorf("release is retired (pair#439) and folds nothing; fold with `write --fold ID --if-hash H`"))
		}
		fmt.Fprintln(stderr, "couch: release is retired and did nothing: the agent holds no lock (pair#439); fold events with `write --fold ID`")
		return 0
	}
	file, err := filepath.Abs(a.file)
	if err != nil {
		return fail(err)
	}
	namespace, err := rt.ResolveNamespace()
	if err != nil {
		if a.operatorLockCommand() {
			return sessionSyncUnavailable(stdout, err)
		}
		return fail(err)
	}
	s := sessionsync.New(namespace.Dir())
	switch a.verb {
	case "operator-acquire", "operator-release":
		request := sessionsync.OperatorLockRequest{File: file, ID: a.id, Turn: a.turn}
		var result sessionsync.OperatorLock
		if a.verb == "operator-acquire" {
			result, err = s.AcquireOperator(request)
		} else {
			result, err = s.ReleaseOperator(request)
		}
		if err != nil {
			return fail(err)
		}
		if err := json.NewEncoder(stdout).Encode(sessionSyncResponse{Code: "ok", OperatorLock: &result}); err != nil {
			return fail(err)
		}
	case "register", "forget", "rename":
		var res sessionsync.RegistryResult
		switch a.verb {
		case "register":
			res, err = s.Register(file, a.owner)
		case "forget":
			res, err = s.Forget(file)
		case "rename":
			res, err = s.Rename(a.oldFile, file)
		}
		if err != nil {
			return fail(err)
		}
		if res.Registration != nil {
			res.Registration.Namespace = namespace.Dir()
		}
		if a.json {
			if err := json.NewEncoder(stdout).Encode(sessionSyncResponse{Code: "ok", Registration: res.Registration, Changed: res.Changed}); err != nil {
				return fail(err)
			}
		} else if res.Registration != nil {
			fmt.Fprintf(stdout, "%s\nowner: %s\nid: %s\n", res.Registration.File, res.Registration.Owner, res.Registration.ID)
		} else {
			fmt.Fprintln(stdout, "forgotten:", file)
		}
	case "lookup":
		res, err := s.Lookup(file)
		if err != nil {
			return fail(err)
		}
		if res != nil {
			res.Namespace = namespace.Dir()
		}
		if a.json {
			if err := json.NewEncoder(stdout).Encode(sessionSyncResponse{Code: "ok", Registration: res}); err != nil {
				return fail(err)
			}
		} else if res != nil {
			fmt.Fprintf(stdout, "%s\nowner: %s\nid: %s\n", res.File, res.Owner, res.ID)
		} else {
			fmt.Fprintln(stdout, "not registered:", file)
		}
	case "list":
		res, err := s.List()
		if err != nil {
			return fail(err)
		}
		if a.json {
			if err := json.NewEncoder(stdout).Encode(sessionSyncResponse{Code: "ok", Registrations: res}); err != nil {
				return fail(err)
			}
		} else {
			for _, entry := range res {
				fmt.Fprintf(stdout, "%s\t%s\n", entry.File, entry.RegisteredOwner)
			}
		}
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
		var r sessionsync.ReadResult
		if a.frontmatter {
			r, err = s.ReadFrontmatter(file)
		} else {
			r, err = s.Read(file, a.section)
		}
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
	case "round":
		res, err := s.Round(file)
		if err != nil {
			return fail(err)
		}
		if err := json.NewEncoder(stdout).Encode(roundResponse{Code: "ok", Round: res.Round, Cleared: res.Cleared}); err != nil {
			return fail(err)
		}
	case "apply":
		raw, err := os.ReadFile(a.bodyFile)
		if err != nil {
			return fail(fmt.Errorf("reading the edits: %w", err))
		}
		edits, err := sessionsync.ParseEdits(raw)
		if err != nil {
			return fail(&sessionsync.Refusal{Exit: sessionsync.ExitRefused, Code: "edits-invalid", Reason: err.Error()})
		}
		res, err := s.Apply(sessionsync.ApplyRequest{File: file, IfHash: a.ifHash, Edits: edits, Accepts: a.accepts, Fold: a.fold})
		if err != nil {
			return fail(err)
		}
		if a.json {
			if err := json.NewEncoder(stdout).Encode(sessionSyncResponse{Code: "ok", Hash: res.Hash}); err != nil {
				return fail(err)
			}
		} else {
			fmt.Fprintf(stdout, "applied %d record(s)\nhash: %s\n", len(edits), res.Hash)
		}
		reportFolds(stdout, stderr, res)
		reportSessionWrite(rt, namespace.Dir(), res.Owner)
	case "write":
		raw, err := os.ReadFile(a.bodyFile)
		if err != nil {
			return fail(fmt.Errorf("reading the body: %w", err))
		}
		res, err := s.Write(sessionsync.WriteRequest{File: file, Section: a.section, Frontmatter: a.frontmatter, Body: string(raw),
			IfHash: a.ifHash, Accepts: a.accepts, Fold: a.fold})
		if err != nil {
			return fail(err)
		}
		if a.frontmatter {
			fmt.Fprintln(stdout, "wrote frontmatter")
		} else if a.section != "" {
			fmt.Fprintf(stdout, "wrote ## %s\n", a.section)
		} else {
			fmt.Fprintln(stdout, "rewrote the file")
		}
		fmt.Fprintln(stdout, "hash:", res.Hash)
		reportFolds(stdout, stderr, res)
		reportSessionWrite(rt, namespace.Dir(), res.Owner)
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
	// Do not omit Registration: lookup's explicit null is confirmed absence.
	Registration  *sessionsync.Registration
	Registrations []sessionsync.Registration
	Changed       bool
	OperatorLock  *sessionsync.OperatorLock `json:",omitempty"`
	// Invalid lists the bad records of an `edits-invalid` apply.
	Invalid []sessionsync.BadEdit `json:",omitempty"`
}

// roundResponse is `round --json`, the contract parley.nvim#383 reads:
// Round is the sidecar (null when there is none), Cleared whether a submit
// of the file came after it.
type roundResponse struct {
	Code    string
	Error   string
	Round   *sessionsync.Round
	Cleared bool
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

// reportSessionWrite is diagnostic evidence after the guarded write committed.
// An absent, older or slow supervisor must never change that write's result.
func reportSessionWrite(rt Runtime, namespace, owner string) {
	req := callerIdentity(rt, "session-write")
	req.Target = owner
	if couchmessage.ValidateRequest(req) != nil {
		return
	}
	socket, err := couchmessage.SocketPath(namespace, "broker")
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	var response couchmessage.Response
	_ = couchmessage.Call(ctx, socket, req, &response)
}

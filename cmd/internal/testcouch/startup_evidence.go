package testcouch

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/zellijexec"
	"github.com/xianxu/pair/cmd/internal/zellijreply"
)

const startupReadBytes = 64 << 10
const startupSourceBytes = 16 << 10
const startupDirectoryEntries = 128
const startupSiblingNames = 32

func startupLogPath(p RootsPlan) string {
	return filepath.Join(p.Mutable["control"], "wrapper-startup.log")
}

type startupIdentity struct {
	PID     int    `json:"actualPID"`
	Scope   string `json:"PAIR_SCOPE_KEY"`
	Tag     string `json:"PAIR_TAG"`
	Session string `json:"PAIR_SESSION_NAME"`
	Nonce   string `json:"PAIR_LAUNCH_NONCE"`
	order   int
}

// selectStartupRecords visits each supplied line once. It never retains generic
// debug rows, argv, prompts or arbitrary environment fields.
func selectStartupRecords(raw []byte) (string, map[string]startupIdentity, int) {
	var selected strings.Builder
	identities := map[string]startupIdentity{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 4096), startupReadBytes+1)
	visited := 0
	for scanner.Scan() {
		visited++
		line := scanner.Text()
		_, rest, ok := strings.Cut(line, "] ")
		if !ok {
			continue
		}
		label, quoted, ok := strings.Cut(rest, ": ")
		if !ok || (label != "STARTUP-identity" && label != "STARTUP-error" && label != "AGENT-READY-write-fail") {
			continue
		}
		payload, err := strconv.Unquote(quoted)
		if err != nil {
			fmt.Fprintf(&selected, "%s unavailable: malformed quoted record\n", label)
			continue
		}
		if label == "STARTUP-identity" {
			var identity startupIdentity
			if err := json.Unmarshal([]byte(payload), &identity); err != nil {
				fmt.Fprintln(&selected, "STARTUP-identity unavailable: malformed identity")
				continue
			}
			// Re-encode the allowlist, never a raw JSON object with extra fields.
			value, _ := json.Marshal(identity)
			fmt.Fprintf(&selected, "%s %s\n", label, value)
			identity.order = visited
			identities[identity.Scope+"\x00"+identity.Tag] = identity
		} else {
			if label == "STARTUP-error" {
				var fact struct {
					Stage     string `json:"stage"`
					ErrorType string `json:"error_type"`
					Errno     int    `json:"errno"`
				}
				if err := json.Unmarshal([]byte(payload), &fact); err != nil {
					fmt.Fprintln(&selected, "STARTUP-error unavailable: malformed error facts")
					continue
				}
				raw, _ := json.Marshal(fact)
				payload = string(raw)
			}
			if len(payload) > startupSourceBytes {
				payload = payload[:startupSourceBytes]
				payload += "<truncated>"
			}
			fmt.Fprintf(&selected, "%s %q\n", label, payload)
		}
	}
	if scanner.Err() != nil {
		fmt.Fprintf(&selected, "wrapper records unavailable: %v\n", scanner.Err())
	}
	return selected.String(), identities, visited
}

// evidenceRoot confines every file open, including intermediate symlinks.
// Native commands separately require physical private sockets below this root.
type evidenceRoot struct {
	root *os.Root
	path string
}

func openEvidenceRoot(path string) (*evidenceRoot, error) {
	physical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}
	if physical != path {
		return nil, errors.New("evidence root is not physical")
	}
	r, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	return &evidenceRoot{r, path}, nil
}
func (r *evidenceRoot) relative(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", errors.New("evidence path is not absolute")
	}
	rel, err := filepath.Rel(r.path, path)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("evidence path escapes private root")
	}
	return rel, nil
}
func (r *evidenceRoot) read(path string, limit int, tail bool) ([]byte, bool, error) {
	rel, err := r.relative(path)
	if err != nil {
		return nil, false, err
	}
	f, err := r.root.OpenFile(rel, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, false, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, false, err
	}
	if !st.Mode().IsRegular() {
		return nil, false, errors.New("evidence source is not a regular file")
	}
	truncated := st.Size() > int64(limit)
	if tail && truncated {
		if _, err = f.Seek(-int64(limit), io.SeekEnd); err != nil {
			return nil, false, err
		}
	}
	b, err := io.ReadAll(io.LimitReader(f, int64(limit)+1))
	if len(b) > limit {
		b = b[:limit]
		truncated = true
	}
	return b, truncated, err
}
func (r *evidenceRoot) readySiblings(path string) ([]string, bool, error) {
	rel, err := r.relative(path)
	if err != nil {
		return nil, false, err
	}
	f, err := r.root.OpenFile(rel, os.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, false, err
	}
	defer f.Close()
	entries, err := f.ReadDir(startupDirectoryEntries + 1)
	if errors.Is(err, io.EOF) {
		err = nil
	}
	names, truncated, _ := selectReadyNames(entries)
	return names, truncated, err
}
func selectReadyNames(entries []os.DirEntry) ([]string, bool, int) {
	names := []string{}
	truncated := len(entries) > startupDirectoryEntries
	visited := 0
	for _, entry := range entries[:min(len(entries), startupDirectoryEntries)] {
		visited++
		name := entry.Name()
		if artifactpath.IsAgentReadyCandidate(name) {
			if len(names) >= startupSiblingNames {
				truncated = true
				continue
			}
			if len(name) > 256 {
				name = name[:256] + "<truncated>"
			}
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, truncated, visited
}

type startupDump func(context.Context, string, string) (string, error)

// Diagnostic work shares one quarter of the caller's remaining budget; the
// original context stays available to stop and reconcile owned resources.
func startupDiagnosticDeadline(now, caller time.Time) (time.Time, error) {
	remaining := caller.Sub(now)
	if remaining <= 0 {
		return time.Time{}, context.DeadlineExceeded
	}
	return now.Add(remaining / 4), nil
}

type startupSource struct{ scope, tag, agent, nonce string }

// selectStartupSources prioritizes exact failure addresses, then the most
// recently emitted private wrapper identities. Each input is visited once;
// sorting never holds more than five candidates and selection retains four.
func selectStartupSources(history []couchcore.LaunchFailure, identities map[string]startupIdentity, agent string) ([]startupSource, bool, int) {
	var sources []startupSource
	seen := map[string]bool{}
	visited := 0
	add := func(scope, tag, kind, nonce string) {
		key := scope + "\x00" + tag
		if scope == "" || tag == "" || seen[key] || len(sources) >= 4 {
			return
		}
		seen[key] = true
		if kind == "" {
			kind = agent
		}
		sources = append(sources, startupSource{scope, tag, kind, nonce})
	}
	for _, failure := range history[:min(len(history), 4)] {
		visited++
		add(failure.Address.RepoScope, string(failure.Address.Tag), failure.Agent, failure.LaunchNonce)
	}
	var latest []startupIdentity
	for _, identity := range identities {
		visited++
		latest = append(latest, identity)
		sort.Slice(latest, func(i, j int) bool { return latest[i].order > latest[j].order })
		if len(latest) > 4 {
			latest = latest[:4]
		}
	}
	for _, identity := range latest {
		add(identity.Scope, identity.Tag, agent, "")
	}
	return sources, len(history) > 4 || len(identities) > 4, visited
}

func (f *Fleet) captureStartupLog() string {
	r, err := openEvidenceRoot(f.Roots.Root)
	if err != nil {
		return fmt.Sprintf("wrapper diagnostics unavailable: %v", err)
	}
	defer r.root.Close()
	raw, truncated, err := r.read(startupLogPath(f.Roots), startupReadBytes, true)
	selected, _, _ := selectStartupRecords(raw)
	if selected == "" {
		selected = "wrapper startup records unavailable\n"
	}
	return fmt.Sprintf("wrapper_log truncated=%t read_error=%v\n%s", truncated, err, selected)
}

func (f *Fleet) captureStartupEvidence(ctx context.Context, dump startupDump) string {
	var out strings.Builder
	queryCtx := ctx
	if caller, bounded := ctx.Deadline(); bounded {
		deadline, e := startupDiagnosticDeadline(time.Now(), caller)
		if e != nil {
			return fmt.Sprintf("startup evidence unavailable: %v", e)
		}
		var cancel context.CancelFunc
		queryCtx, cancel = context.WithDeadline(ctx, deadline)
		defer cancel()
		fmt.Fprintf(&out, "pane queries share diagnostic_deadline=%s\n", deadline.Format(time.RFC3339Nano))
	}
	r, err := openEvidenceRoot(f.Roots.Root)
	if err != nil {
		return fmt.Sprintf("startup evidence unavailable: %v", err)
	}
	defer r.root.Close()
	raw, truncated, err := r.read(startupLogPath(f.Roots), startupReadBytes, true)
	fmt.Fprintf(&out, "wrapper_log truncated=%t read_error=%v\n", truncated, err)
	if err != nil {
		fmt.Fprintln(&out, "wrapper diagnostics unavailable")
	}
	selected, identities, _ := selectStartupRecords(raw)
	if selected == "" {
		fmt.Fprintln(&out, "wrapper startup records unavailable")
	} else {
		out.WriteString(selected)
	}
	// Read the production history only after validating the store below the
	// private descriptor; its production reader also rejects unsafe paths.
	store := f.Roots.Mutable["store"]
	var history []couchcore.LaunchFailure
	if _, _, err := r.read(filepath.Join(store, "launch-diagnostics", "failures.json"), 1, false); err != nil {
		fmt.Fprintf(&out, "failure identities unavailable: %v\n", err)
	} else {
		namespace, e := couchcore.ExistingCouchNamespace(store)
		if e == nil {
			history, e = couchcore.NewLaunchFailureStore(namespace).Read()
		}
		if e != nil {
			fmt.Fprintf(&out, "failure identities unavailable: %v\n", e)
		}
	}
	for _, failure := range history[:min(len(history), 4)] {
		if failure.Address.RepoScope == "" || failure.Address.Tag == "" {
			fmt.Fprintln(&out, "failure source unavailable: incomplete failure address; consulting emitted wrapper identities")
		}
	}
	sources, sourceCut, _ := selectStartupSources(history, identities, f.Options.AgentKind)
	fmt.Fprintf(&out, "startup_sources=%d truncated=%t\n", len(sources), sourceCut)
	census := map[string]bool{}
	for _, source := range sources {
		global := filepath.Join(f.Roots.Mutable["data"], "pair")
		paths, pathErr := artifactpath.Resolve(artifactpath.Address{DataDir: global, RepoScope: source.scope, Tag: source.tag})
		if pathErr != nil {
			fmt.Fprintf(&out, "source unavailable: %v\n", pathErr)
			continue
		}
		fmt.Fprintf(&out, "source scope=%q tag=%q\n", source.scope, source.tag)
		if !census[paths.ScopeDir()] {
			census[paths.ScopeDir()] = true
			names, cut, e := r.readySiblings(paths.ScopeDir())
			fmt.Fprintf(&out, "ready_siblings=%q truncated=%t read_error=%v\n", names, cut, e)
		}
		identity, ok := identities[source.scope+"\x00"+source.tag]
		agent := source.agent
		if agent != "claude" && agent != "codex" {
			fmt.Fprintln(&out, "ready and pane sources unavailable: fixture agent identity unavailable")
			continue
		}
		readyPath, pathErr := paths.AgentReadyChecked(agent)
		if pathErr != nil {
			fmt.Fprintf(&out, "expected ready unavailable: %v\n", pathErr)
			continue
		}
		readyRaw, readyCut, readyErr := r.read(readyPath, startupSourceBytes, false)
		fmt.Fprintf(&out, "expected_ready=%q truncated=%t read_error=%v\n", filepath.Base(readyPath), readyCut, readyErr)
		if readyErr == nil && !readyCut {
			record, decodeErr := readiness.Decode(string(readyRaw))
			if decodeErr != nil {
				fmt.Fprintf(&out, "ready identity unavailable: %v\n", decodeErr)
			} else {
				fmt.Fprintf(&out, "ready_identity tag=%q agent=%q session=%q nonce=%q pid=%d matches_emitted=%t\n", record.Tag, record.Agent, record.Session, record.Nonce, record.PID, ok && record.Tag == source.tag && record.Agent == agent && record.Session == identity.Session && record.Nonce == identity.Nonce)
			}
		} else {
			fmt.Fprintln(&out, "ready identity unavailable")
		}
		if !ok || identity.PID <= 0 || identity.Session == "" || identity.Nonce == "" || strings.Contains(identity.Session, "<truncated>") || strings.ContainsAny(identity.Session, "/\x00\n\r") {
			fmt.Fprintln(&out, "pane dump unavailable: exact emitted launch identity unavailable")
			continue
		}
		if source.nonce != "" && source.nonce != identity.Nonce {
			fmt.Fprintf(&out, "pane dump unavailable: emitted launch nonce mismatch expected=%q observed=%q\n", source.nonce, identity.Nonce)
			continue
		}
		if source.nonce == "" {
			fmt.Fprintln(&out, "pane attribution: current observed identity; failed-attempt attribution unavailable")
		} else {
			fmt.Fprintln(&out, "pane attribution: emitted identity matches recorded launch nonce")
		}
		paneRaw, cut, paneErr := r.read(paths.Pane(agent), 4096, false)
		var pane struct {
			ID string `json:"pane_id"`
		}
		if paneErr == nil {
			paneErr = json.Unmarshal(paneRaw, &pane)
		}
		id, idErr := strconv.Atoi(pane.ID)
		if paneErr != nil || cut || idErr != nil || id < 0 {
			fmt.Fprintf(&out, "pane dump unavailable: pane sidecar truncated=%t read_error=%v id_error=%v\n", cut, paneErr, idErr)
			continue
		}
		if queryCtx.Err() != nil {
			fmt.Fprintf(&out, "pane dump unavailable: shared diagnostic budget: %v\n", queryCtx.Err())
			continue
		}
		if dump == nil {
			dump = f.dumpStartupPane
		}
		text, queryErr := dump(queryCtx, identity.Session, pane.ID)
		cut = len(text) > startupSourceBytes
		if cut {
			text = text[:startupSourceBytes]
		}
		fmt.Fprintf(&out, "pane_dump session=%q pane=%q truncated=%t query_error=%v output=%q\n", identity.Session, pane.ID, cut, queryErr, text)
		if queryErr != nil {
			fmt.Fprintln(&out, "pane dump unavailable: native query failed (rollback may already have removed pane)")
		}
	}
	return out.String()
}

type startupOutput struct {
	data      []byte
	truncated bool
}

func (w *startupOutput) Write(p []byte) (int, error) {
	n := len(p)
	remaining := startupSourceBytes - len(w.data)
	if len(p) > remaining {
		p = p[:remaining]
		w.truncated = true
	}
	w.data = append(w.data, p...)
	return n, nil
}
func (f *Fleet) dumpStartupPane(ctx context.Context, session, pane string) (string, error) {
	id, idErr := strconv.Atoi(pane)
	if session == "" || session == "." || session == ".." || strings.ContainsAny(session, "/\x00\n\r") || strings.Contains(session, "<truncated>") || idErr != nil || id < 0 {
		return "", errors.New("pane query requires an exact emitted session and numeric pane")
	}
	if _, bounded := ctx.Deadline(); !bounded {
		return "", errors.New("pane query requires bounded cleanup context")
	}
	// The native query gets no ambient environment or default socket fallback.
	sockets := f.Roots.Mutable["sockets"]
	r, err := openEvidenceRoot(f.Roots.Root)
	if err != nil {
		return "", err
	}
	defer r.root.Close()
	rel, err := r.relative(sockets)
	if err != nil {
		return "", err
	}
	dir, err := r.root.OpenFile(rel, os.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return "", err
	}
	defer dir.Close()
	physical, err := filepath.EvalSymlinks(sockets)
	if err != nil || physical != sockets {
		return "", errors.New("private socket root is not physical")
	}
	query, cancel := zellijreply.AdmissionContext(ctx)
	defer cancel()
	cmd := exec.CommandContext(query, f.Binaries.Zellij, "-s", session, "action", "dump-screen", "--pane-id", "terminal_"+pane)
	cmd.Env = append([]string(nil), f.Environment...)
	cmd.Env = append(cmd.Env, "ZELLIJ_SOCKET_DIR="+sockets, "ZELLIJ_SESSION_NAME="+session)
	stdout, stderr := &startupOutput{}, &startupOutput{}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	err = zellijexec.Run(cmd)
	if err != nil || len(stderr.data) > 0 || stderr.truncated || stdout.truncated {
		err = errors.Join(err, fmt.Errorf("pane query stdout_truncated=%t stderr_truncated=%t stderr=%q", stdout.truncated, stderr.truncated, stderr.data))
	}
	return string(stdout.data), err
}

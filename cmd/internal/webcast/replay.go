package webcast

import (
	"github.com/xianxu/pair/cmd/internal/castlock"

	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	castassets "github.com/xianxu/pair/cmd/internal/webcast/assets"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// ReplayServer serves the recordings under a root for replay in the viewer
// (#471): GET only, under its token, with the viewer's CSP.
//
//	<token>/                     the recordings, newest first
//	<token>/r/<id>/              the viewer, which plays the recording
//	<token>/r/<id>/records       its records, merged, one JSON line each
//	<token>/r/<id>/assets/<hash> its asset bodies, through the cast's proxy
//	<token>/r/<id>/<module>      the viewer's modules
//
// A recording is read from disk when its records are asked for, so one
// still being written replays as far as it has got. It writes nothing.
//
// One scoped to a recording (NewRecordingServer) serves only that one: no
// listing, and every other id is not found, so its link grants what it
// names (#477).
type ReplayServer struct {
	token string
	root  string
	only  string // the one recording served; "" serves every one and the listing

	mu      sync.Mutex
	proxies map[string]*castassets.Proxy // by recording id; at most one per directory under root
}

// NewReplayServer serves the recordings under root (each a directory).
func NewReplayServer(token, root string) *ReplayServer {
	return &ReplayServer{token: token, root: root, proxies: map[string]*castassets.Proxy{}}
}

// NewRecordingServer serves only the recording id under root.
func NewRecordingServer(token, root, id string) *ReplayServer {
	s := NewReplayServer(token, root)
	s.only = id
	return s
}

// recordingIDShape is the shape of a recording directory's name: one path
// element, no traversal.
var recordingIDShape = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// ValidRecordingID says whether id can name a recording directory.
func ValidRecordingID(id string) bool { return recordingIDShape.MatchString(id) }

// ReplayLink is the URL of a recording at t ms: base is the server's
// <scheme>://<host>/<token>.
func ReplayLink(base, id string, t int64) string {
	return fmt.Sprintf("%s/r/%s/#t=%d", strings.TrimSuffix(base, "/"), id, max(t, 0))
}

func (s *ReplayServer) Owns(segment string) bool { return broadcast.OwnsToken(segment, s.token) }

// LockFor is nil: replay links are served without a code for now; they are
// read-only recordings, and locking them is a follow-up to pair#495.
func (s *ReplayServer) LockFor(string) *castlock.Lock { return nil }

func (s *ReplayServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	broadcast.SetSecurityHeaders(w.Header(), contentSecurityPolicy)
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	segment, rest, ok := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if !ok || !s.Owns(segment) {
		http.NotFound(w, r)
		return
	}
	if rest == "" && s.only == "" {
		s.list(w)
		return
	}
	id, sub, ok := strings.Cut(strings.TrimPrefix(rest, "r/"), "/")
	if !strings.HasPrefix(rest, "r/") || !ok || !s.exists(id) {
		http.NotFound(w, r)
		return
	}
	switch {
	case sub == "records":
		s.records(w, r, id)
	case strings.HasPrefix(sub, "assets/"):
		p, err := s.proxy(id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		p.Serve(w, r, strings.TrimPrefix(sub, "assets/"))
	default:
		serveViewerFile(w, r, sub)
	}
}

// exists says whether id is a recording this server serves.
func (s *ReplayServer) exists(id string) bool {
	return (s.only == "" || id == s.only) && recordingExists(s.root, id)
}

// recordingExists says whether id names a recording directory directly
// under root (not a link out of it).
func recordingExists(root, id string) bool {
	if !recordingIDShape.MatchString(id) {
		return false
	}
	info, err := os.Lstat(filepath.Join(root, id))
	return err == nil && info.IsDir()
}

func (s *ReplayServer) records(w http.ResponseWriter, r *http.Request, id string) {
	rec, err := recording.ReadDir(filepath.Join(s.root, id))
	if err != nil {
		http.Error(w, "the recording can't be read", http.StatusInternalServerError)
		return
	}
	s.bind(id, rec)
	var buf bytes.Buffer
	for _, x := range rec.Records {
		line, err := recording.Encode(x)
		if err != nil {
			http.Error(w, "the recording can't be read", http.StatusInternalServerError)
			return
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}
	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(buf.Bytes())
}

// bind points id's proxy at rec's assets: the asset table as of the
// records the viewer was given.
func (s *ReplayServer) bind(id string, rec *recording.Recording) *castassets.Proxy {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.proxies[id]
	if p == nil {
		p = castassets.NewProxy()
		s.proxies[id] = p
	}
	p.Bind(castassets.Reopen(rec))
	return p
}

// proxy is id's asset proxy, binding it from disk if no records were
// asked for yet.
func (s *ReplayServer) proxy(id string) (*castassets.Proxy, error) {
	s.mu.Lock()
	p := s.proxies[id]
	s.mu.Unlock()
	if p != nil {
		return p, nil
	}
	rec, err := recording.ReadDir(filepath.Join(s.root, id))
	if err != nil {
		return nil, err
	}
	return s.bind(id, rec), nil
}

// serveViewerFile serves one of the viewer's files; "" is its page.
// collab.js is the terminal viewer's (broadcast.CollabJS), imported by the
// notes module, so every server of the viewer serves it.
func serveViewerFile(w http.ResponseWriter, r *http.Request, name string) {
	if name == "collab.js" {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Write(broadcast.CollabJS())
		return
	}
	a, ok := assets[name]
	if !ok {
		http.NotFound(w, r)
		return
	}
	body, err := webFS.ReadFile(a.file)
	if err != nil {
		http.Error(w, "unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", a.contentType)
	w.Write(body)
}

// Listed is one recording in the listing. Duration is known only when
// Ended: a last line whose envelope doesn't parse (a torn write) leaves it
// unknown, not 0.
type Listed struct {
	ID       string
	Started  string
	Duration time.Duration
	Ended    bool
	Notes    int
}

// Length is the listing's length column.
func (l Listed) Length() string {
	if !l.Ended {
		return "unknown"
	}
	return l.Duration.String()
}

// ListRecordings reads what the listing shows of each recording under
// root, newest first: the head's start, the last event's time and the
// number of notes. It reads each events file's first and last lines only.
func ListRecordings(root string) ([]Listed, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var out []Listed
	for _, e := range entries {
		if !e.IsDir() || !recordingIDShape.MatchString(e.Name()) {
			continue
		}
		dir := filepath.Join(root, e.Name())
		head, end, ended, err := headAndEnd(filepath.Join(dir, recording.FileEvents))
		if err != nil {
			continue // not a recording
		}
		out = append(out, Listed{ID: e.Name(), Started: head, Duration: time.Duration(end) * time.Millisecond, Ended: ended, Notes: countLines(filepath.Join(dir, recording.FileNotes))})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Started > out[j].Started })
	return out, nil
}

// headAndEnd reads an events file's start (its head record, the first
// line) and its end (the last record's t), without reading the records in
// between: the last line can be a whole document snapshot, so the file is
// read backwards in chunks to that line's start, and only its envelope's
// leading "t" is read (every record line begins {"t":N, as Encode writes it).
// ended is false when that line doesn't begin so (a torn write).
func headAndEnd(path string) (started string, end int64, ended bool, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, false, err
	}
	defer f.Close()
	first, err := bufio.NewReaderSize(f, 4096).ReadSlice('\n')
	if err != nil && err != io.EOF {
		return "", 0, false, err // a head line is small: a longer first line isn't one
	}
	var head struct {
		K       string `json:"k"`
		Started string `json:"started"`
	}
	if json.Unmarshal(bytes.TrimSpace(first), &head) != nil || head.K != "head" {
		return "", 0, false, errNotRecording
	}
	info, err := f.Stat()
	if err != nil {
		return "", 0, false, err
	}
	start, err := lastLineStart(f, info.Size())
	if err != nil {
		return "", 0, false, err
	}
	lead := make([]byte, 32)
	n, _ := f.ReadAt(lead, start)
	if m := leadingT.FindSubmatch(lead[:n]); m != nil {
		end, err := strconv.ParseInt(string(m[1]), 10, 64)
		return head.Started, end, err == nil, nil
	}
	return head.Started, 0, false, nil
}

var (
	errNotRecording = errors.New("not a recording")
	leadingT        = regexp.MustCompile(`^\{"t":(\d+)[,}]`) // the digits end: a torn "t" is not one
)

// lastLineStart is the offset where the file's last non-empty line begins.
func lastLineStart(f *os.File, size int64) (int64, error) {
	const chunk = 64 * 1024
	end := size
	// Skip trailing newlines.
	for end > 0 {
		b := make([]byte, 1)
		if _, err := f.ReadAt(b, end-1); err != nil {
			return 0, err
		}
		if b[0] != '\n' {
			break
		}
		end--
	}
	for at := end; at > 0; {
		from := max(0, at-chunk)
		buf := make([]byte, at-from)
		if _, err := f.ReadAt(buf, from); err != nil && err != io.EOF {
			return 0, err
		}
		if i := bytes.LastIndexByte(buf, '\n'); i >= 0 {
			return from + int64(i) + 1, nil
		}
		at = from
	}
	return 0, nil
}

func countLines(path string) int {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	return bytes.Count(b, []byte("\n"))
}

var listTemplate = template.Must(template.New("list").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="referrer" content="no-referrer"><title>Couch web cast recordings</title>
<style>body{font:14px/1.5 system-ui,sans-serif;margin:24px 16px;color-scheme:light dark}table{border-collapse:collapse}td,th{padding:4px 12px;text-align:left}td.n{text-align:right;font-variant-numeric:tabular-nums}</style>
</head><body><h1>Recordings</h1>
{{if .}}<table><tr><th>Recording</th><th>Started</th><th>Length</th><th>Notes</th></tr>
{{range .}}<tr><td><a href="r/{{.ID}}/">{{.ID}}</a></td><td>{{.Started}}</td><td class="n">{{.Length}}</td><td class="n">{{.Notes}}</td></tr>
{{end}}</table>{{else}}<p>No recordings yet.</p>{{end}}
</body></html>
`))

func (s *ReplayServer) list(w http.ResponseWriter) {
	recs, err := ListRecordings(s.root)
	if err != nil && !os.IsNotExist(err) {
		http.Error(w, "the recordings can't be read", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	listTemplate.Execute(w, recs)
}

package webcast

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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
type ReplayServer struct {
	token string
	root  string

	mu      sync.Mutex
	proxies map[string]*castassets.Proxy // by recording id; at most one per directory under root
}

// NewReplayServer serves the recordings under root (each a directory).
func NewReplayServer(token, root string) *ReplayServer {
	return &ReplayServer{token: token, root: root, proxies: map[string]*castassets.Proxy{}}
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
	if rest == "" {
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

// exists says whether id names a recording directory directly under root
// (not a link out of it).
func (s *ReplayServer) exists(id string) bool {
	if !recordingIDShape.MatchString(id) {
		return false
	}
	info, err := os.Lstat(filepath.Join(s.root, id))
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
func serveViewerFile(w http.ResponseWriter, r *http.Request, name string) {
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

// Listed is one recording in the listing.
type Listed struct {
	ID       string
	Started  string
	Duration time.Duration
	Notes    int
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
		first, last, err := endLines(filepath.Join(dir, recording.FileEvents))
		if err != nil {
			continue // not a recording
		}
		var head struct {
			K       string `json:"k"`
			Started string `json:"started"`
		}
		var end struct {
			T int64 `json:"t"`
		}
		if json.Unmarshal(first, &head) != nil || head.K != "head" {
			continue
		}
		json.Unmarshal(last, &end)
		out = append(out, Listed{ID: e.Name(), Started: head.Started, Duration: time.Duration(end.T) * time.Millisecond, Notes: countLines(filepath.Join(dir, recording.FileNotes))})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Started > out[j].Started })
	return out, nil
}

// endLines reads a file's first and last non-empty lines.
func endLines(path string) (first, last []byte, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	if !sc.Scan() {
		return nil, nil, io.ErrUnexpectedEOF
	}
	first = bytes.Clone(sc.Bytes())
	info, err := f.Stat()
	if err != nil {
		return nil, nil, err
	}
	const tail = 64 * 1024
	buf := make([]byte, min(info.Size(), tail))
	if _, err := f.ReadAt(buf, info.Size()-int64(len(buf))); err != nil && err != io.EOF {
		return nil, nil, err
	}
	lines := bytes.Split(bytes.TrimRight(buf, "\n"), []byte("\n"))
	return first, lines[len(lines)-1], nil
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
{{range .}}<tr><td><a href="r/{{.ID}}/">{{.ID}}</a></td><td>{{.Started}}</td><td class="n">{{.Duration}}</td><td class="n">{{.Notes}}</td></tr>
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

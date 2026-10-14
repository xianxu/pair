package webcast

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/notedraft"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// WithNoteDrafts binds private draft ownership supplied by the authenticated
// host. A nil/empty owner fails closed; viewer attribution is never authority.
func (s *Server) WithNoteDrafts(notes *notedraft.Service, owner func(*http.Request) string) *Server {
	s.noteDrafts = notes
	s.noteOwner = owner
	return s
}

func (s *Server) noteDraftRoute(w http.ResponseWriter, r *http.Request, rest string, link castLink) bool {
	switch rest {
	case "prepare-note", "draft-status", "save-note", "cancel-draft":
	default:
		return false
	}
	if s.board == nil || s.noteDrafts == nil {
		http.NotFound(w, r)
		return true
	}
	if !link.annotates() || s.noteOwner == nil {
		http.Error(w, refuseViewOnly, http.StatusForbidden)
		return true
	}
	private := s.noteOwner(r)
	if private == "" {
		http.Error(w, refuseViewOnly, http.StatusForbidden)
		return true
	}
	generation, err := strconv.ParseUint(r.Header.Get("X-Couch-Annotation-Generation"), 10, 64)
	if err != nil {
		http.Error(w, refuseAnnotationOff, http.StatusForbidden)
		return true
	}
	admission, err := s.board.board.AdmitAnnotationGeneration(generation)
	if err != nil {
		code, msg := boardAnswer(err)
		http.Error(w, msg, code)
		return true
	}
	body, release, ok := s.board.admit(w, r, (*boardRoutes).allowNote, board.MaxOperatorBody)
	if !ok {
		return true
	}
	defer release()
	if err = admission.Check(); err != nil {
		code, msg := boardAnswer(err)
		http.Error(w, msg, code)
		return true
	}
	segment, _, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
	hash := sha256.Sum256([]byte(private + "\x00" + segment))
	owner := notedraft.Owner{Scope: hex.EncodeToString(hash[:]), Generation: generation}
	var status notedraft.Status
	switch rest {
	case "prepare-note":
		var in struct {
			Request string           `json:"request"`
			Seg     int              `json:"seg"`
			Node    recording.NodeID `json:"node"`
			Box     []*float64       `json:"box"`
			Region  []*float64       `json:"region"`
		}
		if !broadcast.DecodeStrict(body, &in) || in.Request == "" || !draftBox(in.Box) || !draftBox(in.Region) {
			err = board.ErrShape
			break
		}
		var prepared board.PreparedNote
		prepared, err = admission.PrepareNote(board.PrepareNoteIn{Seg: in.Seg, Node: in.Node, Box: recording.Box{*in.Box[0], *in.Box[1], *in.Box[2], *in.Box[3]}, Region: recording.Box{*in.Region[0], *in.Region[1], *in.Region[2], *in.Region[3]}}, false)
		if err == nil {
			status, err = s.noteDrafts.Prepare(r.Context(), owner, in.Request, admission, prepared, false)
		}
	case "save-note":
		var in struct {
			Request string       `json:"request"`
			Draft   string       `json:"draft"`
			Viewer  board.Viewer `json:"viewer"`
			Color   *int         `json:"color"`
			Text    string       `json:"text"`
		}
		if !broadcast.DecodeStrict(body, &in) || in.Color == nil {
			err = board.ErrShape
			break
		}
		status, err = s.noteDrafts.Save(r.Context(), owner, in.Request, in.Draft, admission, in.Viewer, *in.Color, in.Text, nil)
	default:
		var in struct {
			Request string `json:"request"`
			Draft   string `json:"draft"`
		}
		if !broadcast.DecodeStrict(body, &in) {
			err = board.ErrShape
			break
		}
		id := in.Draft
		if id == "" {
			id = in.Request
		}
		if id == "" {
			err = board.ErrShape
			break
		}
		if rest == "draft-status" {
			status, err = s.noteDrafts.Status(owner, id)
		} else {
			err = s.noteDrafts.Cancel(owner, id)
			status = notedraft.Status{State: "cancelled", Request: in.Request, Draft: in.Draft}
		}
	}
	if err != nil {
		code, msg := boardAnswer(err)
		if code == http.StatusServiceUnavailable {
			msg = err.Error()
		}
		http.Error(w, msg, code)
		return true
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
	return true
}

func draftBox(v []*float64) bool {
	if len(v) != 4 {
		return false
	}
	for _, n := range v {
		if n == nil {
			return false
		}
	}
	return true
}

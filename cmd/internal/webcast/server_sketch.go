package webcast

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/drive"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

const drawingEpochHeader = "X-Couch-Drawing-Epoch"

func sketchHeaderInteger(raw string, safe bool) (uint64, error) {
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || strconv.FormatUint(n, 10) != raw || safe && n > recording.MaxSketchInteger {
		return 0, board.ErrShape
	}
	return n, nil
}

// sketchRoute uses the drive link and surviving seat credential, never claimed
// author/owner fields. Label/status retain metadata-only authority after handoff.
func (s *Server) sketchRoute(w http.ResponseWriter, r *http.Request, rest string, link castLink) bool {
	switch rest {
	case "sketch-begin", "sketch-stroke-begin", "sketch-points", "sketch-stroke-end", "sketch-undo", "sketch-redo", "sketch-label", "sketch-status", "sketch-source", "sketch-asset":
	default:
		return false
	}
	w.Header().Set("Cache-Control", "no-store")
	if !link.drives() {
		http.Error(w, refuseViewOnly, http.StatusForbidden)
		return true
	}
	if s.sketches == nil || s.board == nil || s.drive == nil {
		http.NotFound(w, r)
		return true
	}
	epoch, err := sketchHeaderInteger(r.Header.Get(drawingEpochHeader), true)
	if err != nil {
		http.Error(w, "Drawing is unavailable.", http.StatusForbidden)
		return true
	}
	generation, err := sketchHeaderInteger(r.Header.Get("X-Couch-Annotation-Generation"), false)
	if err != nil {
		http.Error(w, refuseAnnotationOff, http.StatusForbidden)
		return true
	}
	annotation, err := s.board.board.AdmitAnnotationGeneration(generation)
	if err != nil {
		sketchBoardError(w, err)
		return true
	}
	key := r.Header.Get(drive.SeatHeader)
	identity, err := s.drive.ctrl.AdmitDrawingMetadata(key)
	if err != nil {
		sketchDriveError(w, err)
		return true
	}
	metadata := rest == "sketch-label" || rest == "sketch-status"
	var drawing drive.DrawingPermit
	if !metadata {
		drawing, err = s.drive.ctrl.AdmitDrawing(epoch, key)
		if err != nil {
			sketchDriveError(w, err)
			return true
		}
	}
	digest := sha256.Sum256([]byte(key))
	a := board.SketchAuthority{Author: identity.Principal(), Owner: hex.EncodeToString(digest[:]), Epoch: epoch, Annotation: annotation, MetadataCheck: func() error {
		if err := r.Context().Err(); err != nil {
			return err
		}
		return identity.Check()
	}}
	if !metadata {
		a.Check = drawing.Check
	}
	limit := (*boardRoutes).allowStroke
	if rest == "sketch-label" {
		limit = (*boardRoutes).allowNote
	}
	body, release, ok := s.board.admit(w, r, limit, board.MaxSketchBody)
	if !ok {
		return true
	}
	defer release()
	command, err := board.ParseSketchCommand(rest, body)
	if err != nil || command.Epoch != epoch {
		http.Error(w, refuseInvalid, http.StatusBadRequest)
		return true
	}
	check := func() error {
		if err := annotation.Check(); err != nil {
			return err
		}
		if err := a.MetadataCheck(); err != nil {
			return err
		}
		if !metadata {
			return drawing.Check()
		}
		return nil
	}
	if err = check(); err != nil {
		sketchDriveError(w, err)
		return true
	}
	switch rest {
	case "sketch-source":
		total, err := s.sketches.SourceLength(a, command.Sketch, command.Source)
		if err != nil {
			sketchBoardError(w, err)
			return true
		}
		if command.Offset > total {
			http.Error(w, refuseInvalid, http.StatusRequestedRangeNotSatisfiable)
			return true
		}
		data, err := s.sketches.ReadSource(a, command.Sketch, command.Source, command.Offset, command.Length)
		if err != nil {
			sketchBoardError(w, err)
			return true
		}
		if confirmed, e := s.sketches.SourceLength(a, command.Sketch, command.Source); e != nil || confirmed != total {
			if e == nil {
				e = board.ErrStale
			}
			sketchBoardError(w, e)
			return true
		}
		if err = check(); err != nil {
			sketchDriveError(w, err)
			return true
		}
		writeSketchChunk(w, command.Source, command.Offset, total, data)
	case "sketch-asset":
		if err = s.sketches.AdmitSourceAsset(a, command.Sketch, command.Source, command.Asset); err != nil {
			sketchBoardError(w, err)
			return true
		}
		if s.sketchAsset == nil {
			http.NotFound(w, r)
			return true
		}
		data, total, err := s.sketchAsset(command.Asset, command.Offset, command.Length)
		if err != nil {
			http.Error(w, "The saved asset is unavailable.", http.StatusNotFound)
			return true
		}
		if len(data) > command.Length || len(data) > board.MaxSketchTransfer || total < 0 || uint64(total) > recording.MaxSketchInteger || command.Offset > total || len(data) != min(command.Length, total-command.Offset) {
			http.Error(w, "The saved asset is unavailable.", http.StatusInternalServerError)
			return true
		}
		if err = s.sketches.AdmitSourceAsset(a, command.Sketch, command.Source, command.Asset); err != nil {
			sketchBoardError(w, err)
			return true
		}
		if err = check(); err != nil {
			sketchDriveError(w, err)
			return true
		}
		writeSketchChunk(w, command.Source, command.Offset, total, data)
	default:
		var reply board.SketchReply
		if rest == "sketch-status" {
			reply = s.sketches.Status(r.Context(), a, command)
		} else {
			reply = s.sketches.Execute(r.Context(), a, command)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(reply)
	}
	return true
}

func sketchDriveError(w http.ResponseWriter, err error) {
	if errors.Is(err, board.ErrDisabled) || errors.Is(err, board.ErrClosed) || errors.Is(err, board.ErrStale) {
		sketchBoardError(w, err)
		return
	}
	status, text := drive.Answer(err)
	http.Error(w, text, status)
}
func sketchBoardError(w http.ResponseWriter, err error) {
	status, text := boardAnswer(err)
	http.Error(w, text, status)
}
func writeSketchChunk(w http.ResponseWriter, source string, offset, total int, data []byte) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Source string `json:"source"`
		Offset int    `json:"offset"`
		Total  int    `json:"total"`
		Body   string `json:"body"`
	}{source, offset, total, base64.StdEncoding.EncodeToString(data)})
}

package sessioninventory

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
)

// The two native latest selectors deliberately have distinct cwd predicates.
// TUI asks SQLite about repository scope; exec inspects the latest turn cwd.
// Source: rust-v0.162.1 tui/src/lib.rs latest_session_lookup_params and
// exec/src/lib.rs resolve_resume_thread_id / parse_latest_turn_context_cwd.
func resolveScopedCodexLatest(ctx context.Context, rt Runtime, q NativeSelectorRequest) (result NativeSelectorResolution, resultErr error) {
	incomplete := func() (NativeSelectorResolution, error) {
		return NativeSelectorResolution{Status: NativeSelectorIncomplete, Reason: "Codex latest evidence is incomplete or requires native fallback"}, ctx.Err()
	}
	scope := q.Config.CodexScope
	if scope == nil || !filepath.IsAbs(scope.CWD) || scope.CWD != q.CWD || scope.Provider == "" {
		return incomplete()
	}
	rootMatches := false
	for _, root := range rt.NativeRoots(AgentCodex) {
		if root.Name == "codex-sessions" && root.Path == filepath.Join(scope.CodexHome, "sessions") {
			rootMatches = true
		}
	}
	if !rootMatches {
		return incomplete()
	}
	queryRT, err := nativeSelectorCodexRuntime(rt, scope.SQLiteHome)
	if err != nil {
		return incomplete()
	}
	bounded, err := newNativeSelectorRuntime(ctx, queryRT)
	if err != nil {
		return incomplete()
	}
	defer func() {
		if closeErr := bounded.Close(); closeErr != nil {
			result = NativeSelectorResolution{Status: NativeSelectorIncomplete, Reason: "native SQLite snapshot cleanup incomplete"}
			resultErr = errors.Join(resultErr, closeErr)
		}
	}()
	var selected codexLatestRow
	switch scope.Mode {
	case CodexModeTUI:
		selected, err = collectCodexTUILatest(bounded, q)
	case CodexModeExec:
		selected, err = collectCodexExecLatest(bounded, q)
	default:
		return incomplete()
	}
	if err != nil {
		return incomplete()
	}
	if selected.id == "" {
		return NativeSelectorResolution{Status: NativeSelectorNone, Reason: "no native database match; native fallback required"}, ctx.Err()
	}
	// Preserve every inspected candidate, including higher rows excluded by cwd,
	// across the inventory's independent listing. A later valid session_meta
	// cannot make an earlier cwd observation current.
	selectionEntries := make([]FileEntry, 0, len(bounded.snapshots))
	for _, entry := range bounded.snapshots {
		selectionEntries = append(selectionEntries, entry)
	}
	validation, diagnostics, err := NewIncrementalInventory(bounded, Catalog{}).ValidateNamedContent(AgentCodex, selected.id)
	for _, entry := range selectionEntries {
		current, snapshotErr := bounded.io.StatRecoveryFile(ctx, entry.Artifact)
		if snapshotErr != nil || !sameRecoveryEntry(entry, current) {
			return incomplete()
		}
	}

	if err := bounded.validateSQLiteSnapshots(); err != nil {
		return incomplete()
	}
	if ctx.Err() != nil {
		return NativeSelectorResolution{}, ctx.Err()
	}
	if err != nil || bounded.failure != nil || validation.State.Disputed || validation.State.Role != RoleRoot || !validation.Fact.Resumable || validation.State.NativeID != selected.id {
		return incomplete()
	}
	for _, d := range diagnostics {
		if d.Code == DiagnosticParentConflict && d.NativeID != nil && *d.NativeID == selected.id {
			return incomplete()
		}
	}
	return NativeSelectorResolution{Status: NativeSelectorResolved, NativeID: selected.id}, nil
}

type codexLatestRow struct {
	id, path, cwd string
	ms            int64
}

func codexSQLQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
func codexLatestRows(r *nativeSelectorRuntime, predicate string, beforeMS int64, limit int) ([]codexLatestRow, error) {
	outputLimit := int64(1 << 20)
	if remaining := RecoveryTotalReadLimit - r.bytes; remaining < outputLimit {
		outputLimit = remaining
	}
	if outputLimit <= 0 || limit > RecoveryMetadataLimit-r.entries {
		return nil, ErrReadLimit
	}
	if beforeMS != 0 {
		predicate += " AND updated_at_ms<" + strconv.FormatInt(beforeMS, 10)
	}
	result, err := r.QuerySQLite(Artifact{StorageRoot: "codex-state", RelativePath: "state_5.sqlite", Kind: ArtifactMetadata}, "SELECT id, rollout_path, updated_at_ms, cwd FROM threads WHERE archived=0 AND preview<>''"+predicate+" ORDER BY updated_at_ms DESC LIMIT "+strconv.Itoa(limit), outputLimit)
	if err != nil {
		return nil, err
	}
	if strings.Join(result.Columns, ",") != "id,rollout_path,updated_at_ms,cwd" || len(result.Rows) > limit {
		return nil, ErrReadLimit
	}
	if len(result.Rows) > RecoveryMetadataLimit-r.entries {
		return nil, ErrReadLimit
	}
	r.entries += len(result.Rows)
	// CSV quotes can double every byte. Charge a conservative encoding bound,
	// including the header, so SQL and transcript reads share one ceiling.
	encodedBytes := int64(64)
	for _, row := range result.Rows {
		for _, value := range row {
			encodedBytes += 2*int64(len(value)) + 3
		}
	}
	if encodedBytes > RecoveryTotalReadLimit-r.bytes {
		return nil, ErrReadLimit
	}
	r.bytes += encodedBytes
	rows := make([]codexLatestRow, 0, len(result.Rows))
	for _, row := range result.Rows {
		if len(row) != 4 {
			return nil, ErrReadLimit
		}
		ms, err := strconv.ParseInt(row[2], 10, 64)
		if err != nil || ms <= 0 {
			return nil, ErrReadLimit
		}
		rows = append(rows, codexLatestRow{id: row[0], path: row[1], ms: ms, cwd: row[3]})
	}
	return rows, nil
}
func collectCodexTUILatest(r *nativeSelectorRuntime, q NativeSelectorRequest) (codexLatestRow, error) {
	sources := "'cli','vscode'"
	if q.IncludeNonInteractive {
		sources += ",'exec','mcp'"
	}
	predicate := " AND source IN (" + sources + ") AND model_provider=" + codexSQLQuote(q.Config.CodexScope.Provider)
	if !q.All {
		cwds := q.Config.CodexScope.CWDs
		if len(cwds) == 0 {
			cwds = []string{q.CWD}
		}
		quoted := make([]string, 0, len(cwds))
		for _, cwd := range cwds {
			quoted = append(quoted, codexSQLQuote(cwd))
		}
		predicate += " AND cwd IN (" + strings.Join(quoted, ",") + ")"
	}
	rows, err := codexLatestRows(r, predicate, 0, 2)
	if err != nil || len(rows) == 0 {
		return codexLatestRow{}, err
	}
	if len(rows) > 1 && rows[0].ms == rows[1].ms {
		return codexLatestRow{}, errors.New("ambiguous native activity")
	}
	artifact, err := codexLatestArtifact(r, rows[0])
	if err != nil {
		return codexLatestRow{}, err
	}
	if err = validateCodexLatestFirst(r, artifact, rows[0].id); err != nil {
		return codexLatestRow{}, err
	}
	return rows[0], nil
}
func collectCodexExecLatest(r *nativeSelectorRuntime, q NativeSelectorRequest) (codexLatestRow, error) {
	predicate := " AND model_provider=" + codexSQLQuote(q.Config.CodexScope.Provider)
	var chosen codexLatestRow
	var previous int64
	groupSize := 0
	// Native state reads page_size+1 rows, then anchors UpdatedAt strictly
	// below the last emitted row. It supplies no ID tie breaker.
	var cursor int64
	for emitted := 0; emitted < RecoveryMetadataLimit; emitted += 100 {
		rows, err := codexLatestRows(r, predicate, cursor, 101)
		if err != nil {
			return codexLatestRow{}, err
		}
		page := rows
		if len(page) > 100 {
			page = page[:100]
		}
		for _, row := range page {
			if previous != 0 && row.ms > previous {
				return codexLatestRow{}, ErrArtifactChanged
			}
			if previous == row.ms {
				groupSize++
			} else {
				groupSize = 1
			}
			previous = row.ms
			if chosen.id != "" {
				if row.ms == chosen.ms {
					return codexLatestRow{}, errors.New("ambiguous native activity")
				}
				return chosen, nil
			}
			artifact, err := codexLatestArtifact(r, row)
			if err != nil {
				return codexLatestRow{}, err
			}
			if err = validateCodexLatestFirst(r, artifact, row.id); err != nil {
				return codexLatestRow{}, err
			}
			match := q.All
			if !match {
				cwd, err := latestCodexTurnCWD(r, artifact)
				if err != nil {
					return codexLatestRow{}, err
				}
				if cwd == "" {
					cwd = row.cwd
				}
				physical, err := r.io.ResolveRecoveryCWD(r.ctx, cwd)
				if err != nil {
					return codexLatestRow{}, err
				}
				match = physical == q.CWD
			}
			if match {
				if groupSize > 1 {
					return codexLatestRow{}, errors.New("ambiguous native activity")
				}
				chosen = row
			}
		}
		if len(rows) <= 100 {
			return chosen, nil
		}
		// Advancing a timestamp-only cursor across this tie could hide a candidate
		// with the same rank. Keep native compatibility instead of inventing order.
		if rows[99].ms == rows[100].ms {
			return codexLatestRow{}, errors.New("ambiguous native page boundary")
		}
		if chosen.id != "" {
			return chosen, nil
		}
		cursor = rows[99].ms
	}
	return codexLatestRow{}, ErrReadLimit
}
func codexLatestArtifact(r *nativeSelectorRuntime, row codexLatestRow) (Artifact, error) {
	if !IsCanonicalCodexSessionID(row.id) || !filepath.IsAbs(row.path) {
		return Artifact{}, ErrPathEscape
	}
	for _, root := range r.NativeRoots(AgentCodex) {
		if root.Name != "codex-sessions" {
			continue
		}
		rel, err := filepath.Rel(root.Path, row.path)
		if err != nil {
			continue
		}
		a := Artifact{StorageRoot: root.Name, RelativePath: filepath.ToSlash(rel), Kind: ArtifactTranscript}
		id, ok := codexPathID(a.RelativePath)
		if !validArtifact(a) || !ok || id != row.id {
			continue
		}
		if _, err = r.StatFile(a); err != nil {
			return Artifact{}, err
		}
		return a, nil
	}
	return Artifact{}, ErrPathEscape
}
func validateCodexLatestFirst(r *nativeSelectorRuntime, a Artifact, id string) error {
	e := r.snapshots[targetArtifactKey(a)]
	var raw []byte
	for offset := int64(0); offset < e.Size && len(raw) <= RecoveryRecordLimit; {
		chunk, _, err := r.ReadAt(a, offset, readChunkSize)
		if err != nil {
			return err
		}
		if len(chunk) == 0 {
			return ErrArtifactChanged
		}
		raw = append(raw, chunk...)
		offset += int64(len(chunk))
		end := bytes.IndexByte(raw, '\n')
		if end < 0 && offset < e.Size {
			continue
		}
		if end >= 0 {
			raw = raw[:end]
		}
		if len(raw) > RecoveryRecordLimit {
			return ErrReadLimit
		}
		if !codexNativeSessionMetaEligible(raw) {
			return ErrArtifactChanged
		}
		state, diagnostics, err := ValidateCodexDelta(e, nil, []FramedJSONLRecord{{Bytes: raw}})
		if err != nil || !state.FirstRecordValidated || state.Disputed || state.NativeID != id {
			return ErrArtifactChanged
		}
		for _, d := range diagnostics {
			if d.Code == DiagnosticParentConflict {
				return ErrArtifactChanged
			}
		}
		return nil
	}
	return ErrReadLimit
}

// latestCodexTurnCWD only establishes absence after reaching the beginning of
// an unchanged file. A partial tail or exhausted reverse scan remains unknown.
func latestCodexTurnCWD(r *nativeSelectorRuntime, a Artifact) (string, error) {
	e := r.snapshots[targetArtifactKey(a)]
	end := e.Size
	var pending []byte
	var consumed int64
	finish := func(cwd string) (string, error) {
		after, err := r.io.StatRecoveryFile(r.ctx, a)
		if err != nil {
			return "", err
		}
		if !sameRecoveryEntry(e, after) {
			return "", ErrArtifactChanged
		}
		return cwd, nil
	}
	parse := func(raw []byte) (string, error) {
		if len(bytes.TrimSpace(raw)) == 0 {
			return "", nil
		}
		if len(raw) > RecoveryRecordLimit {
			return "", ErrReadLimit
		}
		var row struct {
			Type    string `json:"type"`
			Payload struct {
				CWD string `json:"cwd"`
			} `json:"payload"`
		}
		if err := json.Unmarshal(raw, &row); err != nil {
			return "", err
		}
		if row.Type == "turn_context" {
			cwd, eligible := codexNativeTurnContextCWD(raw)
			if !eligible {
				return "", ErrArtifactChanged
			}
			return cwd, nil
		}
		return "", nil
	}
	for end > 0 {
		n := readChunkSize
		if n > end {
			n = end
		}
		if consumed+n > 32<<20 {
			return "", ErrReadLimit
		}
		start := end - n
		chunk, _, err := r.ReadAt(a, start, n)
		if err != nil {
			return "", err
		}
		if int64(len(chunk)) != n {
			return "", ErrArtifactChanged
		}
		consumed += n
		end = start
		pending = append(chunk, pending...)
		for {
			i := bytes.LastIndexByte(pending, '\n')
			if i < 0 {
				break
			}
			line := pending[i+1:]
			cwd, err := parse(line)
			if err != nil {
				return "", err
			}
			if cwd != "" {
				return finish(cwd)
			}
			pending = pending[:i]
		}
		if len(pending) > RecoveryRecordLimit {
			return "", ErrReadLimit
		}
	}
	cwd, err := parse(pending)
	if err != nil {
		return "", err
	}
	return finish(cwd)
}

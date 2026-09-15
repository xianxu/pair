package terminalqualify

// Coverage preserves the original integration obligation inventory. Cases only
// replaces a placeholder when an actual production-seam experiment exists;
// the remaining placeholders never count as successful qualification.
func Coverage() []Case {
	const plan = "workshop/plans/000255-terminal-abstraction-plan.md#behavioral-contract"
	gaps := []struct{ id, capability, reason string }{
		{"device-attributes", "device attributes and terminfo agreement", "DA replies advertise a specific terminal model/capability set; the production profile and its matching DA1/DA2/XTGETTCAP replies must be chosen together before testing truthfulness."},
		{"composition-switch", "chrome isolation and A/B switching", "Both production consumers must exercise distinct margins/save slots, split switch boundaries and evicted replay through the shared compositor; endpoint fixtures alone cannot establish adoption."},
		{"hidden-origin", "hidden endpoint query/effect origin", "Endpoint FIFO tests establish independent origin-bound replies; selected presenter and consumer routing still require composed verification."},
		{"reply-backpressure-routing", "ordered replies and operator events", "Candidate drain tests prove local teardown; production bounded writer, origin routing and byte ordering remain unimplemented."},
		{"sync-publication", "synchronized frame publication", "The candidate exposes mutable screen state, not an immutable publishable frame boundary."},
		{"sync-recovery", "bounded synchronized-output recovery", "No production publication timeout or recovery policy exists to exercise."},
		{"parser-memory-bound", "bounded parser retention", "Fixed 16KiB malformed/unterminated fixtures cannot prove a memory ceiling for an unending string or parameter stream."},
		{"wrapper-composition", "actual wrapper stream through Zellij", "Must transport real stdoutChunk/stripCodexOutputMarkers, notification rewriting, Return translation and query tracking output through isolated Zellij, preserving raw/transformed observer semantics."},
		{"clipboard-policy", "clipboard reads and one-shot writes", "Endpoint callbacks establish typed writes and deterministic unsupported reads; preserving product policy and outer delivery requires consumer integration qualification."},
		{"notification-origin", "Pair notification effects", "Endpoint typed notification origin is exercised; product focus-at-delivery, attention marking and once-only outer delivery require consumer integration."},
		{"partial-parent-write", "presentation failure and input admission", "No presenter exists to force delayed frames, partial writes, paused admission and release ordering."},
		{"drag-destination", "drag continuity across switch/close", "Endpoint event encodings cannot prove press-owned destination or switch/close cancellation policy."},
		{"terminfo-profile", "truthful terminfo and capability advertisement", "The pair-vt-256color profile and local queries are implemented; actual packaged child environment and shell/nvim/Zellij compatibility remain consumer/native obligations."},
		{"live-display-selection", "sustained live drawing and selection", "M4 requires operator acceptance of continuous highlights and absence of display corruption across both consumers and reattachment."},
	}
	cases := make([]Case, 0, len(gaps))
	for _, g := range gaps {
		cases = append(cases, Case{ID: g.id, Capability: g.capability, Source: plan, Uncovered: g.reason})
	}
	return cases
}

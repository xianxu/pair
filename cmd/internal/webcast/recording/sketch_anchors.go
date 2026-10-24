package recording

import "slices"

// HasAnchor proves recorded node identity only within the sketch's source
// document. Point geometry, current visibility and page text are not identity.
func (s *SketchState) HasAnchor(document int, node NodeID) bool {
	return s.anchors[document][node]
}

// AnchorNodes returns an owned, stable list. Consumers cache it by source
// document and AnchorRevision; point appends never rebuild or scan this index.
func (s *SketchState) AnchorNodes(document int) []NodeID {
	nodes := make([]NodeID, 0, len(s.anchors[document]))
	for node := range s.anchors[document] {
		nodes = append(nodes, node)
	}
	slices.Sort(nodes)
	return nodes
}
func (s *SketchState) AnchorRevision() uint64 { return s.anchorRevision }
func (s *SketchState) addAnchor(document int, node NodeID) {
	if s.HasAnchor(document, node) {
		return
	}
	if s.anchors == nil {
		s.anchors = map[int]map[NodeID]bool{}
	}
	if s.anchors[document] == nil {
		s.anchors[document] = map[NodeID]bool{}
	}
	s.anchors[document][node] = true
	s.anchorRevision++
}

// SketchAnchorNodesOnPageLoad includes anchors from earlier activations of the
// same load. Recorded handles still require a live backend identity witness.
func (s *State) SketchAnchorNodesOnPageLoad(load *PageLoad) []NodeID {
	seen := map[NodeID]bool{}
	for doc, anchors := range s.Sketches.anchors {
		if !s.OnPageLoad(doc, load) {
			continue
		}
		for node := range anchors {
			seen[node] = true
		}
	}
	nodes := make([]NodeID, 0, len(seen))
	for node := range seen {
		nodes = append(nodes, node)
	}
	slices.Sort(nodes)
	return nodes
}

// SketchAnchorsOnPageLoad checks only the requested batch against existing
// source-document buckets in O(D*q) work for D documents and q requested IDs.
// Chrome's fixed batch limit makes this O(B*D) over B delivery batches, alongside
// the preexisting O(B*(H+DOM)) feedback/backend scans; D is bounded by retained
// anchor history H. No full-anchor copy/sort is repeated here. The result and
// allocation are bounded by q; the full sorted projection remains available to
// the existing once-per-revision cache.
func (s *State) SketchAnchorsOnPageLoad(load *PageLoad, wanted map[NodeID]bool) map[NodeID]bool {
	found := make(map[NodeID]bool, len(wanted))
	if len(wanted) == 0 || load.Validate() != nil {
		return found
	}
	for document, anchors := range s.Sketches.anchors {
		if !s.OnPageLoad(document, load) {
			continue
		}
		for node, requested := range wanted {
			if requested && anchors[node] {
				found[node] = true
			}
		}
	}
	return found
}

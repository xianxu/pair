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

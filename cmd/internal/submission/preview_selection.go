package submission

// SelectSketchPreview is the single oldest-first batch policy used by private
// preview and preparation. Only exact dispatched/resolved versions are omitted.
func SelectSketchPreview(candidates []PreviewItem, delivered map[PreviewItem]struct{}, stamp PreviewStamp) PreviewSelection {
	out := PreviewSelection{Stamp: stamp}
	for _, v := range candidates {
		if _, sent := delivered[v]; sent {
			continue
		}
		if len(out.Items) < 8 {
			out.Items = append(out.Items, v)
		} else {
			out.Remaining++
		}
	}
	return out
}

package sessioninventory

// ChosenFilenameHandshake acknowledges a Pair-generated ID only when exactly
// one recognized root filename first appears after the complete launch baseline.
// It intentionally does not interpret agent-owned metadata fields.
func ChosenFilenameHandshake(agent Agent, nativeID string, baseline []TargetArtifactBoundary, observations []ArtifactObservation) *Artifact {
	if nativeID == "" {
		return nil
	}
	old := map[string]bool{}
	for _, b := range baseline {
		old[b.StorageRoot+"\x00"+b.RelativePath] = true
	}
	var result *Artifact
	for _, o := range observations {
		if o.Agent != agent || observationNativeID(agent, o.Entry.Artifact) != nativeID {
			continue
		}
		if old[targetArtifactKey(o.Entry.Artifact)] || result != nil {
			return nil
		}
		artifact := o.Entry.Artifact
		artifact.Kind = ArtifactTranscript
		result = &artifact
	}
	return result
}

package testcouch

type Resource struct{ ID, Generation, Kind string }
type OwnedRun struct {
	Phase, Error string
	Resources    map[string]Resource
}
type RunEvent struct {
	Kind, Error string
	Resource    Resource
}

// ReduceRun retains exact incarnations until an outcome for that incarnation
// confirms absence. Late starts stay owned even after the creation gate closes.
func ReduceRun(s OwnedRun, e RunEvent) OwnedRun {
	next := s
	next.Resources = make(map[string]Resource, len(s.Resources)+1)
	for k, v := range s.Resources {
		next.Resources[k] = v
	}
	key := e.Resource.ID + "\x00" + e.Resource.Generation
	switch e.Kind {
	case "own":
		if e.Resource.ID == "" || e.Resource.Generation == "" {
			next.Phase = "stuck"
			next.Error = "resource lacks incarnation"
			return next
		}
		next.Resources[key] = e.Resource
		if next.Phase == "" {
			next.Phase = "ready"
		}
		if next.Phase == "cleaned" {
			next.Phase = "closing"
		}
	case "closing":
		next.Phase = "closing"
	case "gone":
		delete(next.Resources, key)
	case "unknown":
		next.Phase = "stuck"
		next.Error = e.Error
	}
	if (next.Phase == "closing" || next.Phase == "stuck") && e.Kind != "unknown" && len(next.Resources) == 0 {
		next.Phase = "cleaned"
		next.Error = ""
	}
	return next
}

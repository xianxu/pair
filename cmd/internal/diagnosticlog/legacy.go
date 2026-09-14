package diagnosticlog

import (
	"errors"
	"os"
)

// PreviewLegacy accepts only an exact diagnostic-family path already classified
// by its owning inventory. It is not discovery authority for external paths.
// Old diagnostic mtime is generation-age evidence, not session-use evidence.
func previewLegacyCurrent(path string, options Options) ([]Segment, error) {
	p, e := canonical(path)
	if e != nil {
		return nil, e
	}
	options = normalized(options)
	if _, e := os.Lstat(directory(p)); e == nil {
		return Preview(p, options, 100)
	} else if !os.IsNotExist(e) {
		return nil, e
	}
	st, e := regular(p)
	if e != nil {
		return nil, e
	}
	reason := ""
	if options.Proof == nil {
		reason = ErrUnknownWriters.Error()
	} else if e = options.Proof(p, nil); e != nil {
		reason = e.Error()
	}
	eligible := reason == "" && DecideSegment(options.Now(), st.ModTime())
	if reason == "" && !eligible {
		reason = "within seven-day retention"
	}
	return []Segment{{Path: p, Bytes: st.Size(), LastWrite: st.ModTime(), Eligible: eligible, Reason: reason}}, nil
}

// CollectLegacy upgrades only eligible, proven stopped diagnostics into the
// managed journal protocol. Young/unknown files get no metadata side effects.
func adoptLegacy(path string, options Options) (bool, error) {
	p, e := canonical(path)
	if e != nil {
		return false, e
	}
	options = normalized(options)
	if _, e := os.Lstat(directory(p)); e == nil {
		return true, nil
	} else if !os.IsNotExist(e) {
		return false, e
	}
	rows, e := previewLegacyCurrent(p, options)
	if e != nil {
		return false, e
	}
	if len(rows) != 1 || !rows[0].Eligible {
		return false, nil
	}
	before, e := regular(p)
	if e != nil {
		return false, e
	}
	expected := generation{Identity: fileIdentity(before), Start: options.Now().UTC(), LastWrite: before.ModTime(), Size: before.Size(), ModTime: before.ModTime()}
	e = locked(p, true, func() error {
		// Another upgraded writer may have adopted it since preview; use its
		// managed state and clocks during the final collection decision.
		if _, e := load(p); e == nil {
			return nil
		} else if !os.IsNotExist(e) {
			return e
		}
		st, e := regular(p)
		if e != nil {
			return e
		}
		if e = matches(st, expected); e != nil {
			return e
		}
		if options.Proof == nil {
			return ErrUnknownWriters
		}
		if e = options.Proof(p, nil); e != nil {
			return e
		}
		if !DecideSegment(options.Now(), st.ModTime()) {
			return errors.New("legacy generation changed age")
		}
		if options.Registry != nil {
			if e = options.Registry(RegistryEntry{Version: 1, Path: p, Directory: directory(p), Lock: lockPath(p)}); e != nil {
				return e
			}
		}
		return save(p, diskState{Version: 1, Path: p, Current: expected}, true)
	})
	if e != nil {
		return false, e
	}
	return true, nil
}

func PreviewLegacy(path string, o Options) ([]Segment, error) {
	rows, _, done, e := PreviewLegacyPage(path, o, "", 100)
	if e == nil && !done {
		e = ErrMore
	}
	return rows, e
}
func CollectLegacy(path string, o Options) ([]Segment, error) {
	rows, _, done, e := CollectLegacyPage(path, o, "", 100)
	if e == nil && !done {
		e = ErrMore
	}
	return rows, e
}

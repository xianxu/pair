package launcher

import "path/filepath"

type legacyImportPair struct {
	src string
	dst string
}

func legacyImportPlan(tag, globalDataDir, scopedDataDir string, exists func(string) bool) []legacyImportPair {
	var pairs []legacyImportPair
	for _, src := range renamePathsFor(tag, globalDataDir) {
		if !exists(src) {
			continue
		}
		rel, err := filepath.Rel(globalDataDir, src)
		if err != nil {
			continue
		}
		dst := filepath.Join(scopedDataDir, rel)
		if exists(dst) {
			continue
		}
		pairs = append(pairs, legacyImportPair{src: src, dst: dst})
	}
	return pairs
}

func importLegacyFlatTag(rt Runtime, tag, globalDataDir, scopedDataDir string) bool {
	if globalDataDir == "" || scopedDataDir == "" || globalDataDir == scopedDataDir {
		return false
	}
	imported := false
	for _, pair := range legacyImportPlan(tag, globalDataDir, scopedDataDir, func(p string) bool {
		_, ok := rt.FileSize(p)
		return ok
	}) {
		if copyLegacyPath(rt, pair.src, pair.dst) {
			imported = true
		}
	}
	if copyLegacyPath(rt, filepath.Join(globalDataDir, "queue-"+tag), filepath.Join(scopedDataDir, "queue-"+tag)) {
		imported = true
	}
	return imported
}

func copyLegacyPath(rt Runtime, src, dst string) bool {
	if raw, err := rt.ReadFile(src); err == nil {
		if _, ok := rt.FileSize(dst); ok {
			return false
		}
		return rt.WriteAtomic(dst, raw) == nil
	}
	entries, err := rt.ReadDir(src)
	if err != nil {
		return false
	}
	imported := false
	for _, rel := range entries {
		target := filepath.Join(dst, rel)
		if _, ok := rt.FileSize(target); ok {
			continue
		}
		raw, err := rt.ReadFile(filepath.Join(src, rel))
		if err != nil {
			continue
		}
		if err := rt.WriteAtomic(target, raw); err == nil {
			imported = true
		}
	}
	return imported
}

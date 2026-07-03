package runtimebundle

import "testing"

func TestManifestDigestIsOrderIndependent(t *testing.T) {
	a := RuntimeManifest{
		Assets: []RuntimeAsset{
			{Path: "bin/pair-wrap", Mode: 0o755, Size: 3, Digest: "aaa"},
			{Path: "nvim/init.lua", Mode: 0o644, Size: 4, Digest: "bbb"},
		},
	}
	b := RuntimeManifest{
		Assets: []RuntimeAsset{
			{Path: "nvim/init.lua", Mode: 0o644, Size: 4, Digest: "bbb"},
			{Path: "bin/pair-wrap", Mode: 0o755, Size: 3, Digest: "aaa"},
		},
	}

	gotA, err := a.ManifestDigest()
	if err != nil {
		t.Fatalf("ManifestDigest(a) error = %v", err)
	}
	gotB, err := b.ManifestDigest()
	if err != nil {
		t.Fatalf("ManifestDigest(b) error = %v", err)
	}
	if gotA == "" {
		t.Fatal("ManifestDigest() = empty")
	}
	if gotA != gotB {
		t.Fatalf("digest differs by order: %q != %q", gotA, gotB)
	}
}

func TestManifestRejectsUnsafePaths(t *testing.T) {
	tests := []struct {
		name   string
		assets []RuntimeAsset
	}{
		{name: "empty", assets: []RuntimeAsset{{Path: "", Mode: 0o644, Digest: "a"}}},
		{name: "absolute", assets: []RuntimeAsset{{Path: "/bin/pair-wrap", Mode: 0o755, Digest: "a"}}},
		{name: "dotdot", assets: []RuntimeAsset{{Path: "bin/../pair-wrap", Mode: 0o755, Digest: "a"}}},
		{name: "duplicate", assets: []RuntimeAsset{
			{Path: "bin/pair-wrap", Mode: 0o755, Digest: "a"},
			{Path: "bin/pair-wrap", Mode: 0o755, Digest: "a"},
		}},
		{name: "empty digest", assets: []RuntimeAsset{{Path: "bin/pair-wrap", Mode: 0o755}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RuntimeManifest{Assets: tt.assets}.Validate()
			if err == nil {
				t.Fatal("Validate() error = nil, want error")
			}
		})
	}
}

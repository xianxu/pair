package couchmessage

import (
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"io"
	"os"
)

// BuildIdentityOfFile hashes the executable at path and reads its embedded
// Go build info (#421). The hash is the identity; a binary without build info
// still has one, with an empty revision.
func BuildIdentityOfFile(path string) (BuildIdentity, error) {
	f, err := os.Open(path)
	if err != nil {
		return BuildIdentity{}, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return BuildIdentity{}, err
	}
	id := BuildIdentity{SHA256: hex.EncodeToString(h.Sum(nil))}
	if info, err := buildinfo.ReadFile(path); err == nil {
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				if len(s.Value) <= maxBuildField {
					id.Revision = s.Value
				}
			case "vcs.modified":
				id.Modified = s.Value == "true"
			}
		}
	}
	return id, nil
}

package couchcore

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/slugline"
)

// slugMaxBytes bounds one suggestion read. A slug is one short line; anything
// longer is not one pair-slug wrote.
const slugMaxBytes = 1024

// OSSlugReader reads a thread's pair-slug SUGGESTION (pair#372), never the
// draft-mirrored slug-<tag>: the focus view shows what the agent is doing, not
// the operator's local edits of draft line 1.
type OSSlugReader struct{ DataDir string }

// Read returns the suggestion unfenced (`<branch> | <focus>`). A thread that
// has none yet, or a file that is not a valid slug line, reads as "".
func (r OSSlugReader) Read(ctx context.Context, address ThreadAddress) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: r.DataDir, RepoScope: address.RepoScope, Tag: string(address.Tag)})
	if err != nil {
		return "", err
	}
	file, err := openSwitchRegularFile(paths.SlugProposed())
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, slugMaxBytes+1))
	if err != nil {
		return "", err
	}
	if len(raw) > slugMaxBytes {
		return "", errors.New("slug suggestion exceeds size limit")
	}
	line := strings.TrimSpace(string(raw))
	if !slugline.Valid(line) {
		return "", nil
	}
	return slugline.Unfenced(line), nil
}

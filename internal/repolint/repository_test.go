package repolint

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"slices"
	"strconv"
	"strings"
)

// repositoryRead exposes only the facts consumed by catalog admission. It does
// not expose a Git command interpreter or mutation operations.
type repositoryRead interface {
	TrackedPaths() ([]string, error)
	MainTracking() (mainTracking, error)
	RemoteMainRefs() ([]string, error)
	MergeBase(string, string) (string, error)
	PathsAt(string) ([]string, error)
	ReadBlobs(string, []string) (map[string][]byte, error)
	PathExisted(string, string) (bool, error)
	IsAncestor(string, string) (bool, error)
}
type mainTracking struct {
	Configured    bool
	Remote, Merge string
}
type gitRepository struct{ root string }

// run is the sole subprocess boundary, including historical batched reads.
func (g gitRepository) run(input []byte, args ...string) ([]byte, error) {
	command := exec.Command("git", args...)
	command.Dir = g.root
	command.Stdin = bytes.NewReader(input)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	out, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("git %v: %w: %s", args, err, stderr.String())
	}
	return out, nil
}

// Existing disposable-Git fixture setup uses the same adapter boundary.
func gitOutput(root string, args ...string) ([]byte, error) {
	return (gitRepository{root: root}).run(nil, args...)
}
func (g gitRepository) TrackedPaths() ([]string, error) {
	data, err := g.run(nil, "ls-files", "-z", "--cached")
	if err != nil {
		return nil, err
	}
	return nulPaths(data)
}
func nulPaths(data []byte) ([]string, error) {
	if len(data) == 0 {
		return []string{}, nil
	}
	if data[len(data)-1] != 0 {
		return nil, fmt.Errorf("unterminated NUL path inventory")
	}
	return strings.Split(string(data[:len(data)-1]), "\x00"), nil
}
func (g gitRepository) MainTracking() (mainTracking, error) {
	data, err := g.run(nil, "config", "--get-regexp", `^branch\.main\.(remote|merge)$`)
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 1 {
			return mainTracking{}, err
		}
	}
	config := mainTracking{}
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, " ")
		if !ok || seen[key] {
			return config, fmt.Errorf("ambiguous main tracking configuration")
		}
		seen[key] = true
		config.Configured = true
		switch key {
		case "branch.main.remote":
			config.Remote = strings.TrimSpace(value)
		case "branch.main.merge":
			config.Merge = strings.TrimSpace(value)
		default:
			return config, fmt.Errorf("unexpected main tracking field %q", key)
		}
	}
	return config, nil
}
func (g gitRepository) RemoteMainRefs() ([]string, error) {
	data, err := g.run(nil, "for-each-ref", "--format=%(refname)", "refs/remotes/")
	if err != nil {
		return nil, err
	}
	refs := []string{}
	for _, ref := range strings.Fields(string(data)) {
		if strings.HasSuffix(ref, "/main") {
			refs = append(refs, ref)
		}
	}
	return refs, nil
}
func (g gitRepository) MergeBase(left, right string) (string, error) {
	data, err := g.run(nil, "merge-base", "--all", left, right)
	if err != nil {
		return "", err
	}
	bases := strings.Fields(string(data))
	if len(bases) != 1 {
		return "", fmt.Errorf("need one unambiguous merge base, found %d", len(bases))
	}
	return bases[0], nil
}

type treeBlob struct{ kind, oid string }

func (g gitRepository) tree(revision string) (map[string]treeBlob, error) {
	data, err := g.run(nil, "ls-tree", "-r", "-z", revision, "--")
	if err != nil {
		return nil, err
	}
	rows, err := nulPaths(data)
	if err != nil {
		return nil, err
	}
	tree := map[string]treeBlob{}
	for _, row := range rows {
		metadata, name, ok := strings.Cut(row, "\t")
		fields := strings.Fields(metadata)
		if !ok || name == "" || len(fields) != 3 {
			return nil, fmt.Errorf("invalid historical tree row %q", row)
		}
		if _, err := hex.DecodeString(fields[2]); err != nil || (len(fields[2]) != 40 && len(fields[2]) != 64) {
			return nil, fmt.Errorf("invalid tree object ID")
		}
		if _, duplicate := tree[name]; duplicate {
			return nil, fmt.Errorf("duplicate historical path %q", name)
		}
		tree[name] = treeBlob{fields[1], fields[2]}
	}
	return tree, nil
}
func (g gitRepository) PathsAt(revision string) ([]string, error) {
	tree, err := g.tree(revision)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(tree))
	for name := range tree {
		paths = append(paths, name)
	}
	slices.Sort(paths)
	return paths, nil
}
func (g gitRepository) ReadBlobs(revision string, paths []string) (map[string][]byte, error) {
	// Pathnames are NUL-delimited in ls-tree. Only verified hexadecimal object
	// IDs enter classic cat-file's newline protocol, requiring no newer Git flag.
	tree, err := g.tree(revision)
	if err != nil {
		return nil, err
	}
	out := map[string][]byte{}
	if len(paths) == 0 {
		return out, nil
	}
	var input strings.Builder
	for _, name := range paths {
		blob, ok := tree[name]
		if !ok || blob.kind != "blob" {
			return nil, fmt.Errorf("missing blob %s:%q", revision, name)
		}
		fmt.Fprintln(&input, blob.oid)
	}
	data, err := g.run([]byte(input.String()), "cat-file", "--batch")
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(bytes.NewReader(data))
	for _, name := range paths {
		header, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		fields := strings.Fields(header)
		if len(fields) != 3 || fields[0] != tree[name].oid || fields[1] != "blob" {
			return nil, fmt.Errorf("invalid batch blob header %q", header)
		}
		size, err := strconv.Atoi(fields[2])
		if err != nil || size < 0 || size > len(data) {
			return nil, fmt.Errorf("invalid batch blob size")
		}
		source := make([]byte, size)
		if _, err := io.ReadFull(reader, source); err != nil {
			return nil, err
		}
		delimiter, err := reader.ReadByte()
		if err != nil || delimiter != '\n' {
			return nil, fmt.Errorf("invalid batch blob separator")
		}
		out[name] = source
	}
	if _, err := reader.ReadByte(); err != io.EOF {
		return nil, fmt.Errorf("unexpected trailing batch bytes")
	}
	return out, nil
}
func (g gitRepository) PathExisted(revision, path string) (bool, error) {
	data, err := g.run(nil, "log", "-1", "--format=%H", "--full-history", revision, "--", path)
	if err != nil {
		return false, err
	}
	return len(bytes.TrimSpace(data)) > 0, nil
}
func (g gitRepository) IsAncestor(older, newer string) (bool, error) {
	_, err := g.run(nil, "merge-base", "--is-ancestor", older, newer)
	if err == nil {
		return true, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return false, nil
	}
	return false, err
}

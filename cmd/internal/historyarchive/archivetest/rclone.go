package archivetest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	archive "github.com/xianxu/pair/cmd/internal/historyarchive"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type FakeRemoteObject struct {
	ID    string
	Data  []byte
	IsDir bool
}

// FakeRclone models retained remote bytes, duplicate names, and ambiguous late
// copy outcomes through the production adapter's command seam. Configure only
// between sequential calls; Before may mutate state synchronously.
// pair:m5-concept integration
type FakeRclone struct {
	Objects                                   map[string][]FakeRemoteObject
	FolderID                                  string
	MissingHashes, MissingTool, MissingRemote bool
	Before                                    func(context.Context, string, string) error
	AfterCopyError                            error
	PartialCopyBytes                          int
	FailAfterObjects                          int
	Invocations, CopyBatches, HashBatches     int
	Lists, Copies, Downloads                  int
	nextID                                    int
}

func (f *FakeRclone) Run(ctx context.Context, args []string, limit int) ([]byte, error) {
	f.Invocations++
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.MissingTool {
		return nil, errors.New("rclone unavailable")
	}
	if f.MissingRemote {
		return nil, errors.New("rclone remote unavailable")
	}
	if len(args) < 2 {
		return nil, errors.New("fake rclone missing arguments")
	}
	operation := args[0]
	name := ""
	if f.Before != nil {
		if err := f.Before(ctx, operation, name); err != nil {
			return nil, err
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var output []byte
	switch operation {
	case "lsjson":
		directories := false
		for _, a := range args {
			if a == "--dirs-only" {
				directories = true
			}
		}
		var entries []archive.RemoteEntry
		if directories {
			id := f.FolderID
			if id == "" {
				id = "folder-1"
			}
			entries = []archive.RemoteEntry{{Name: "pair-history", ID: id, IsDir: true, Size: -1}}
		} else {
			f.Lists++
			entries = []archive.RemoteEntry{}
			names := make([]string, 0, len(f.Objects))
			for name := range f.Objects {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				for _, obj := range f.Objects[name] {
					entry := archive.RemoteEntry{Name: name, ID: obj.ID, Size: int64(len(obj.Data)), IsDir: obj.IsDir}
					if obj.IsDir {
						entry.Size = -1
					}
					if !f.MissingHashes && !obj.IsDir {
						sum := sha256.Sum256(obj.Data)
						entry.Hashes = map[string]string{"SHA-256": hex.EncodeToString(sum[:])}
					}
					entries = append(entries, entry)
				}
			}
		}
		output, _ = json.Marshal(entries)
	case "copy":
		f.CopyBatches++
		if len(args) < 3 {
			return nil, errors.New("copy missing destination")
		}
		entries, err := os.ReadDir(args[1])
		if err != nil {
			return nil, err
		}
		copied := 0
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
				return nil, errors.New("fake expects flat regular staging")
			}
			name := entry.Name()
			if len(f.Objects[name]) != 0 {
				continue
			}
			data, err := os.ReadFile(filepath.Join(args[1], name))
			if err != nil {
				return nil, err
			}
			if f.Objects == nil {
				f.Objects = map[string][]FakeRemoteObject{}
			}
			f.nextID++
			if f.PartialCopyBytes > 0 && f.PartialCopyBytes < len(data) {
				data = data[:f.PartialCopyBytes]
			}
			f.Objects[name] = []FakeRemoteObject{{ID: fmt.Sprintf("object-%d", f.nextID), Data: append([]byte(nil), data...)}}
			f.Copies++
			copied++
			if f.FailAfterObjects > 0 && copied >= f.FailAfterObjects {
				return nil, errors.New("partial batch copy failed")
			}
		}
		if f.AfterCopyError != nil {
			return nil, f.AfterCopyError
		}
	case "hashsum":
		f.HashBatches++
		listPath := ""
		for i, arg := range args {
			if arg == "--files-from-raw" && i+1 < len(args) {
				listPath = args[i+1]
			}
		}
		if listPath == "" {
			return nil, errors.New("missing raw file list")
		}
		raw, err := os.ReadFile(listPath)
		if err != nil {
			return nil, err
		}
		var lines strings.Builder
		for name := range strings.SplitSeq(strings.TrimSuffix(string(raw), "\n"), "\n") {
			if name == "" {
				continue
			}
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			objects := f.Objects[name]
			if len(objects) != 1 {
				return nil, errors.New("missing or ambiguous checksum object")
			}
			f.Downloads++
			sum := sha256.Sum256(objects[0].Data)
			fmt.Fprintf(&lines, "%s  %s\n", hex.EncodeToString(sum[:]), name)
		}
		output = []byte(lines.String())
	default:
		return nil, errors.New("unsupported fake rclone command")
	}
	if len(output) > limit {
		return nil, errors.New("rclone output exceeds bound")
	}
	return output, ctx.Err()
}

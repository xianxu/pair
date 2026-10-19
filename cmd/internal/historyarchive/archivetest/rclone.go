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
	"path"
	"sort"
	"strings"
)

type FakeRemoteObject struct {
	ID   string
	Data []byte
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
	Lists, Copies, Downloads                  int
	nextID                                    int
}

func (f *FakeRclone) Run(ctx context.Context, args []string, limit int) ([]byte, error) {
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
	switch operation {
	case "copyto":
		if len(args) < 3 {
			return nil, errors.New("copy missing destination")
		}
		_, object, ok := strings.Cut(args[2], ":")
		if !ok {
			return nil, errors.New("missing remote prefix")
		}
		name = path.Base(object)
	case "hashsum":
		if len(args) < 3 {
			return nil, errors.New("hash missing object")
		}
		_, object, ok := strings.Cut(args[2], ":")
		if !ok {
			return nil, errors.New("missing remote prefix")
		}
		name = path.Base(object)
	}
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
					entry := archive.RemoteEntry{Name: name, ID: obj.ID, Size: int64(len(obj.Data))}
					if !f.MissingHashes {
						sum := sha256.Sum256(obj.Data)
						entry.Hashes = map[string]string{"SHA-256": hex.EncodeToString(sum[:])}
					}
					entries = append(entries, entry)
				}
			}
		}
		output, _ = json.Marshal(entries)
	case "copyto":
		f.Copies++
		data, err := os.ReadFile(args[1])
		if err != nil {
			return nil, err
		}
		if len(f.Objects[name]) != 0 {
			return nil, nil
		}
		if f.Objects == nil {
			f.Objects = map[string][]FakeRemoteObject{}
		}
		f.nextID++
		if f.PartialCopyBytes > 0 && f.PartialCopyBytes < len(data) {
			data = data[:f.PartialCopyBytes]
		}
		f.Objects[name] = []FakeRemoteObject{{ID: fmt.Sprintf("object-%d", f.nextID), Data: append([]byte(nil), data...)}}
		if f.AfterCopyError != nil {
			return nil, f.AfterCopyError
		}
	case "hashsum":
		f.Downloads++
		objects := f.Objects[name]
		if len(objects) != 1 {
			return nil, errors.New("missing or ambiguous checksum object")
		}
		sum := sha256.Sum256(objects[0].Data)
		output = []byte(hex.EncodeToString(sum[:]) + "  " + name + "\n")
	default:
		return nil, errors.New("unsupported fake rclone command")
	}
	if len(output) > limit {
		return nil, errors.New("rclone output exceeds bound")
	}
	return output, ctx.Err()
}

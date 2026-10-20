package archivetest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	archive "github.com/xianxu/pair/cmd/internal/historyarchive"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type FakeRemoteObject struct {
	ID    string
	Data  []byte
	IsDir bool
}

// FakeRclone models retained remote bytes, duplicate names, and ambiguous late
// copy outcomes through the production adapter's command seam. Configure only
// between sequential calls; Before may mutate state synchronously. Configure
// FolderObjects or ChildFolders (even an empty map) to enable hierarchy. An
// Objects-only fixture retains legacy single-folder behavior for arbitrary pins.
// pair:m5-concept integration
type FakeRclone struct {
	Objects                                     map[string][]FakeRemoteObject
	FolderID                                    string
	FolderObjects                               map[string]map[string][]FakeRemoteObject
	ChildFolders                                map[string]map[string][]FakeRemoteObject
	FolderLists                                 map[string]int
	BeforeFolder                                func(context.Context, string, string) error
	Mkdirs                                      int
	AfterMkdirError                             error
	MissingHashes, MissingTool, MissingRemote   bool
	Before                                      func(context.Context, string, string) error
	AfterCopyError                              error
	PartialCopyBytes                            int
	FailAfterObjects                            int
	Invocations, CopyBatches, HashBatches       int
	Lists, Copies, Downloads                    int
	DownloadBatches                             int
	DownloadedLabelBytes, DownloadedCipherBytes int64
	nextID                                      int
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
	folderID := f.rootID()
	pinned := false
	for i, arg := range args {
		if arg == "--drive-root-folder-id" && i+1 < len(args) {
			folderID, pinned = args[i+1], true
		}
	}
	if f.BeforeFolder != nil {
		if err := f.BeforeFolder(ctx, operation, folderID); err != nil {
			return nil, err
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	objects := f.folderObjects(folderID)
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
		if directories && !pinned {
			id := f.rootID()
			entries = []archive.RemoteEntry{{Name: "pair-history", ID: id, IsDir: true, Size: -1}}
		} else {
			f.Lists++
			if f.FolderLists == nil {
				f.FolderLists = map[string]int{}
			}
			f.FolderLists[folderID]++
			entries = []archive.RemoteEntry{}
			names := make([]string, 0, len(objects)+len(f.ChildFolders[folderID]))
			for name := range objects {
				names = append(names, name)
			}
			for name := range f.ChildFolders[folderID] {
				if _, exists := objects[name]; !exists {
					names = append(names, name)
				}
			}
			sort.Strings(names)
			for _, name := range names {
				all := append(append([]FakeRemoteObject(nil), objects[name]...), f.ChildFolders[folderID][name]...)
				for _, obj := range all {
					if directories && !obj.IsDir {
						continue
					}
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
		if len(args) >= 3 && !filepath.IsAbs(args[1]) && filepath.IsAbs(args[2]) {
			return nil, f.downloadBatch(ctx, args, objects)
		}
		if len(args) < 3 || !filepath.IsAbs(args[1]) || filepath.IsAbs(args[2]) {
			return nil, errors.New("unsupported fake copy direction")
		}
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
			if len(objects[name]) != 0 {
				continue
			}
			data, err := os.ReadFile(filepath.Join(args[1], name))
			if err != nil {
				return nil, err
			}
			if objects == nil {
				objects = f.makeFolderObjects(folderID)
			}
			f.nextID++
			if f.PartialCopyBytes > 0 && f.PartialCopyBytes < len(data) {
				data = data[:f.PartialCopyBytes]
			}
			objects[name] = []FakeRemoteObject{{ID: fmt.Sprintf("object-%d", f.nextID), Data: append([]byte(nil), data...)}}
			f.Copies++
			copied++
			if f.FailAfterObjects > 0 && copied >= f.FailAfterObjects {
				return nil, errors.New("partial batch copy failed")
			}
		}
		if f.AfterCopyError != nil {
			return nil, f.AfterCopyError
		}
	case "mkdir":
		if !pinned {
			return nil, errors.New("fake mkdir requires pinned parent")
		}
		_, child, ok := strings.Cut(args[1], ":")
		if !ok || child == "" || child == "." || child == ".." || path.Base(child) != child || strings.ContainsAny(child, "\\\r\n\x00") {
			return nil, errors.New("fake mkdir requires one relative child")
		}
		existing := append(append([]FakeRemoteObject(nil), objects[child]...), f.ChildFolders[folderID][child]...)
		for _, object := range existing {
			if !object.IsDir {
				return nil, errors.New("fake mkdir conflicts with file")
			}
		}
		f.Mkdirs++
		if len(existing) == 0 {
			if f.ChildFolders == nil {
				f.ChildFolders = map[string]map[string][]FakeRemoteObject{}
			}
			if f.ChildFolders[folderID] == nil {
				f.ChildFolders[folderID] = map[string][]FakeRemoteObject{}
			}
			f.nextID++
			f.ChildFolders[folderID][child] = []FakeRemoteObject{{ID: fmt.Sprintf("created-folder-%d", f.nextID), IsDir: true}}
		}
		if f.AfterMkdirError != nil {
			return nil, f.AfterMkdirError
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
			objects := objects[name]
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

// downloadBatch models the flags the production selected-object download uses.
// Transfer budgets count actual bytes written, including retained partial data.
func (f *FakeRclone) downloadBatch(ctx context.Context, args []string, objects map[string][]FakeRemoteObject) error {
	f.DownloadBatches++
	flags := map[string]string{}
	for i := 3; i < len(args); i += 2 {
		if i+1 >= len(args) {
			return errors.New("missing fake download flag value")
		}
		key := args[i]
		if _, ok := flags[key]; ok {
			return errors.New("duplicate fake download flag")
		}
		switch key {
		case "--filter-from", "--transfers", "--max-transfer", "--cutoff-mode", "--max-size", "--drive-root-folder-id", "--retries", "--low-level-retries":
		default:
			return errors.New("unsupported fake download flag")
		}
		flags[key] = args[i+1]
	}
	if flags["--filter-from"] == "" || flags["--transfers"] != "8" || flags["--cutoff-mode"] != "hard" {
		return errors.New("missing bounded fake download flags")
	}
	parseBytes := func(key string) (int64, error) {
		value := flags[key]
		if !strings.HasSuffix(value, "B") {
			return 0, errors.New("fake download limit requires bytes")
		}
		n, err := strconv.ParseInt(strings.TrimSuffix(value, "B"), 10, 64)
		if err != nil || n <= 0 {
			return 0, errors.New("invalid fake download byte limit")
		}
		return n, nil
	}
	budget, err := parseBytes("--max-transfer")
	if err != nil {
		return err
	}
	maxSize, err := parseBytes("--max-size")
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(flags["--filter-from"])
	if err != nil {
		return err
	}
	lines := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
	if len(lines) < 2 || len(lines) > 100001 || lines[len(lines)-1] != "- **" {
		return errors.New("fake download filter must exclude unselected objects")
	}
	copied := 0
	seen := map[string]bool{}
	for _, rule := range lines[:len(lines)-1] {
		if !strings.HasPrefix(rule, "+ /") {
			return errors.New("unsupported fake download filter")
		}
		name := strings.TrimPrefix(rule, "+ /")
		id := strings.TrimSuffix(name, ".tar.zst.gpg")
		if id == name {
			id = strings.TrimSuffix(name, ".json")
		}
		if decoded, err := hex.DecodeString(id); err != nil || len(decoded) != 32 || id == name || id != strings.ToLower(id) || seen[name] {
			return errors.New("invalid fake download object name")
		}
		seen[name] = true
		if err = ctx.Err(); err != nil {
			return err
		}
		if name == "" || filepath.Base(name) != name || strings.ContainsAny(name, "/\\\r\n\x00") {
			return errors.New("invalid fake download name")
		}
		objects := objects[name]
		if len(objects) != 1 {
			return errors.New("missing or ambiguous download object")
		}
		data := objects[0].Data
		if int64(len(data)) > maxSize {
			continue
		}
		clipped := false
		if int64(len(data)) > budget {
			data = data[:budget]
			clipped = true
		}
		if f.PartialCopyBytes > 0 && f.PartialCopyBytes < len(data) {
			data = data[:f.PartialCopyBytes]
		}
		path := filepath.Join(args[2], name)
		file, openErr := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if openErr != nil {
			return openErr
		}
		n, writeErr := io.Copy(file, bytes.NewReader(data))
		closeErr := file.Close()
		budget -= n
		if strings.HasSuffix(name, ".json") {
			f.DownloadedLabelBytes += n
		} else {
			f.DownloadedCipherBytes += n
		}
		f.Downloads++
		copied++
		if err = errors.Join(writeErr, closeErr); err != nil {
			return err
		}
		if clipped {
			return errors.New("max transfer limit reached")
		}
		if f.FailAfterObjects > 0 && copied >= f.FailAfterObjects {
			return errors.New("partial batch download failed")
		}
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	return f.AfterCopyError
}

// Objects is the legacy root store; FolderObjects contains only other pinned folders.
func (f *FakeRclone) rootID() string {
	if f.FolderID != "" {
		return f.FolderID
	}
	return "folder-1"
}
func (f *FakeRclone) folderObjects(id string) map[string][]FakeRemoteObject {
	if id == f.rootID() || (f.FolderObjects == nil && f.ChildFolders == nil) {
		return f.Objects
	}
	return f.FolderObjects[id]
}
func (f *FakeRclone) makeFolderObjects(id string) map[string][]FakeRemoteObject {
	objects := map[string][]FakeRemoteObject{}
	if id == f.rootID() || (f.FolderObjects == nil && f.ChildFolders == nil) {
		f.Objects = objects
	} else {
		if f.FolderObjects == nil {
			f.FolderObjects = map[string]map[string][]FakeRemoteObject{}
		}
		f.FolderObjects[id] = objects
	}
	return objects
}

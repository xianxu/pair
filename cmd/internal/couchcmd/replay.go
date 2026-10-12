package couchcmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/webcast"
)

// replayFlag replays broadcast-web recordings in the viewer (#471). It
// serves on loopback until interrupted; mounting the replay on the shared
// broadcast host inside a running Couch is a follow-up.
const replayFlag = "--broadcast-web-replay"

// replayArgs is `couch --broadcast-web-replay [ID|DIR] [--at T]`.
type replayArgs struct {
	target string
	at     int64
	hasAt  bool
}

func parseReplayArgs(args []string) (replayArgs, error) {
	var a replayArgs
	for i := 0; i < len(args); i++ {
		switch v := args[i]; {
		case v == "--at":
			if i+1 >= len(args) {
				return a, errors.New("--at needs a time in ms")
			}
			i++
			t, err := strconv.ParseInt(args[i], 10, 64)
			if err != nil || t < 0 {
				return a, fmt.Errorf("--at %q: a time in ms since the recording started", args[i])
			}
			a.at, a.hasAt = t, true
		case strings.HasPrefix(v, "-"):
			return a, fmt.Errorf("unknown flag %s", v)
		case a.target != "":
			return a, errors.New("one recording at a time")
		default:
			a.target = v
		}
	}
	if a.hasAt && a.target == "" {
		return a, errors.New("--at needs a recording")
	}
	return a, nil
}

// resolveReplay is the root to serve and the recording to open: nothing
// means every recording in Couch's store; a path (one with a separator, or
// a directory that exists) is that recording; anything else is a
// recording's id in the store.
func resolveReplay(target, recordings string) (root, id string, err error) {
	if target == "" {
		return recordings, "", nil
	}
	if info, statErr := os.Stat(target); strings.ContainsRune(target, os.PathSeparator) || (statErr == nil && info.IsDir()) {
		abs, err := filepath.Abs(target)
		if err != nil {
			return "", "", err
		}
		if info, err := os.Stat(abs); err != nil || !info.IsDir() {
			return "", "", fmt.Errorf("%s is not a recording directory", target)
		}
		root, id = filepath.Dir(abs), filepath.Base(abs)
		if !webcast.ValidRecordingID(id) {
			return "", "", fmt.Errorf("%s: a recording directory's name is letters, digits, '.', '_' and '-'", target)
		}
		return root, id, nil
	}
	if !webcast.ValidRecordingID(target) {
		return "", "", fmt.Errorf("%q is not a recording id", target)
	}
	if info, err := os.Stat(filepath.Join(recordings, target)); err != nil || !info.IsDir() {
		return "", "", fmt.Errorf("no recording %q in %s", target, recordings)
	}
	return recordings, target, nil
}

// runReplay serves the replay on a loopback port until ctx ends, printing
// the link: the listing, or the recording (at --at).
func runReplay(ctx context.Context, args []string, store string, stdout, stderr io.Writer) int {
	a, err := parseReplayArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "couch %s: %v\n", replayFlag, err)
		return 2
	}
	if a.target == "" && store == "" {
		fmt.Fprintf(stderr, "couch %s: Couch has no store directory; name a recording directory\n", replayFlag)
		return 2
	}
	root, id, err := resolveReplay(a.target, recordingsDir(store))
	if err != nil {
		fmt.Fprintf(stderr, "couch %s: %v\n", replayFlag, err)
		return 1
	}
	token, err := broadcast.NewToken()
	if err != nil {
		fmt.Fprintf(stderr, "couch %s: %v\n", replayFlag, err)
		return 1
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(stderr, "couch %s: %v\n", replayFlag, err)
		return 1
	}
	srv := &http.Server{Handler: webcast.NewReplayServer(token, root), ReadHeaderTimeout: 10 * time.Second}
	served := make(chan error, 1)
	go func() { served <- srv.Serve(l) }()
	base := fmt.Sprintf("http://%s/%s", l.Addr(), token)
	if id == "" {
		fmt.Fprintln(stdout, base+"/")
	} else {
		fmt.Fprintln(stdout, webcast.ReplayLink(base, id, a.at))
	}
	fmt.Fprintln(stderr, "Serving on this machine only; Ctrl-C ends it.")
	select {
	case <-ctx.Done():
	case err := <-served:
		fmt.Fprintf(stderr, "couch %s: %v\n", replayFlag, err)
		return 1
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	srv.Shutdown(shutdown)
	return 0
}

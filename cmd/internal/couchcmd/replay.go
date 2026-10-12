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
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/webcast"
	"github.com/xianxu/pair/cmd/internal/webcast/query"
)

// replayFlag replays broadcast-web recordings in the viewer (#471). A
// recording is shared through the running Couch's broadcast Host (#477), so
// its link opens on another device; with no Couch running, or --local, it is
// served on loopback until interrupted.
const replayFlag = "--broadcast-web-replay"

// replayArgs is `couch --broadcast-web-replay [ID|DIR] [--at T] [--local]`.
type replayArgs struct {
	target string
	at     int64
	hasAt  bool
	local  bool
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
		case v == "--local":
			a.local = true
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
		// The real directory: the server serves only directories under its
		// root, never a link out of it, so a link given here is resolved.
		abs, err := filepath.Abs(target)
		if err == nil {
			abs, err = filepath.EvalSymlinks(abs)
		}
		if err != nil {
			return "", "", fmt.Errorf("%s is not a recording directory", target)
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

// runReplay prints a recording's replay link: shared by the running Couch
// when one answers, else served on a loopback port until ctx ends. The
// listing, and --local, are always served on loopback.
func runReplay(ctx context.Context, args []string, store string, call messageCall, clock slotPollClock, stdout, stderr io.Writer) int {
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
	if id != "" && !a.local && store != "" {
		link, err := shareReplay(ctx, store, root, id, a.at, call, clock)
		switch {
		case err == nil:
			fmt.Fprintln(stdout, link.url)
			if link.local {
				fmt.Fprintln(stderr, tunnelOffNote)
			}
			return 0
		case cantShare(err):
			fmt.Fprintf(stderr, "%v.\n", err)
		default:
			fmt.Fprintf(stderr, "couch %s: %v\n", replayFlag, err)
			return 1
		}
	}
	return serveReplay(ctx, root, id, a.at, stdout, stderr)
}

// serveReplay serves root on a loopback port until ctx ends, printing the
// link: the listing, or recording id at at.
func serveReplay(ctx context.Context, root, id string, at int64, stdout, stderr io.Writer) int {
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
		fmt.Fprintln(stdout, webcast.ReplayLink(base, id, at))
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

var (
	errNoCouch             = errors.New("no running Couch answered")
	errCouchPredatesReplay = errors.New("the running Couch predates sharing replays")
	errNotInStore          = errors.New("not in Couch's recordings")
)

// cantShare says sharing was never possible (no Couch, an older one, a
// recording it doesn't hold), as against a share that failed: the replay
// CLI then serves on loopback, and the query CLI refuses.
func cantShare(err error) bool {
	return errors.Is(err, errNoCouch) || errors.Is(err, errCouchPredatesReplay) || errors.Is(err, errNotInStore)
}

const tunnelOffNote = "Couch's tunnel is off (COUCH_BROADCAST_TUNNEL=off): the link works on this machine only."

// Sharing polls every replayPoll, and gives up at replayWait: past the
// Host's probe (broadcast.DefaultProbeTimeout), so a link that will answer
// does before then.
const (
	replayPoll = 300 * time.Millisecond
	replayWait = 45 * time.Second
)

type replayLink struct {
	url   string
	local bool // the tunnel is off: the link works on Couch's machine only
}

// shareReplay asks the running Couch to share recording id (#477) and polls
// until its link answers: the replay link at at. The recording must be in
// Couch's store, which is what Couch shares from.
func shareReplay(ctx context.Context, store, root, id string, at int64, call messageCall, clock slotPollClock) (replayLink, error) {
	if !sameDir(root, recordingsDir(store)) {
		return replayLink{}, fmt.Errorf("%w: Couch shares only %s", errNotInStore, recordingsDir(store))
	}
	socket, err := couchmessage.SocketPath(store, "broker")
	if err != nil {
		return replayLink{}, err
	}
	ask := func(op string) (couchmessage.ReplayShare, error) {
		var res couchmessage.Response
		if err := call(ctx, socket, couchmessage.Request{Op: op, Target: id}, &res); err != nil {
			return couchmessage.ReplayShare{}, fmt.Errorf("%w (%v)", errNoCouch, err)
		}
		switch {
		case predatesOp(res):
			return couchmessage.ReplayShare{}, errCouchPredatesReplay
		case res.Code != "ok" || res.Replay == nil:
			return couchmessage.ReplayShare{}, fmt.Errorf("%s: %s", res.Code, res.Error)
		}
		return *res.Replay, nil
	}
	share, err := ask("replay-share")
	deadline := clock.now().Add(replayWait)
	for err == nil && share.State == string(webcast.ShareOpening) {
		if !clock.now().Before(deadline) {
			return replayLink{}, fmt.Errorf("the link is still opening in Couch; run this again to print it")
		}
		clock.sleep(replayPoll)
		if ctx.Err() != nil {
			return replayLink{}, ctx.Err()
		}
		share, err = ask("replay-status")
	}
	switch {
	case err != nil:
		return replayLink{}, err
	case share.State != string(webcast.ShareReady):
		return replayLink{}, fmt.Errorf("the share is %s: %s", share.State, share.Error)
	}
	return replayLink{url: webcast.ReplayLink(share.Base, id, at), local: share.Local}, nil
}

// replayLinker is the query CLI's link verb: an agent's link to a moment,
// from the running Couch only. An agent's command must return, so it never
// serves on loopback itself. Stdout carries the link alone; a note goes to
// stderr.
func replayLinker(store func() (string, error), call messageCall, clock slotPollClock, stderr io.Writer) query.Linker {
	return func(dir string, at int64) (string, error) {
		st, err := store()
		if err != nil {
			return "", err
		}
		root, id, err := resolveReplay(dir, recordingsDir(st))
		if err != nil {
			return "", err
		}
		link, err := shareReplay(context.Background(), st, root, id, at, call, clock)
		if cantShare(err) {
			return "", fmt.Errorf("%w: `couch %s %s` serves it on this machine", err, replayFlag, dir)
		}
		if err == nil && link.local {
			fmt.Fprintln(stderr, tunnelOffNote)
		}
		return link.url, err
	}
}

// sameDir says whether a and b name one directory, through links.
func sameDir(a, b string) bool {
	if ra, err := filepath.EvalSymlinks(a); err == nil {
		a = ra
	}
	if rb, err := filepath.EvalSymlinks(b); err == nil {
		b = rb
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

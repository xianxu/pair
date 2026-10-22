package webcast

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	castassets "github.com/xianxu/pair/cmd/internal/webcast/assets"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
	"github.com/xianxu/pair/cmd/internal/webcast/sketchimage"
)

// NewSketchImageRenderer renders only already-read recording data. The browser
// has no live-tab connection, and its sandbox can fetch only the private replay.
func NewSketchImageRenderer(chromeBin string) sketchimage.ImageRenderer {
	return &sketchImageRenderer{chromeBin: chromeBin}
}

type sketchImageRenderer struct {
	chromeBin string
	active    sync.Mutex
}

// sketchImageContext bounds the whole serial batch, including CLI callers
// without an inherited preparation deadline. Cleanup retains its own5+1s grace.
func sketchImageContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, 30*time.Second)
}

func (r *sketchImageRenderer) Render(ctx context.Context, b sketchimage.Batch) (images []sketchimage.Image, err error) {
	ctx, cancel := sketchImageContext(ctx)
	defer cancel()
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	if !r.active.TryLock() {
		return nil, errors.New("sketch image: renderer already active")
	}
	defer r.active.Unlock()
	plans, err := sketchimage.ValidateBatch(b)
	if err != nil {
		return nil, err
	}
	server, err := serveSketchImages(b, plans)
	if err != nil {
		return nil, err
	}
	defer func() {
		if e := server.close(); e != nil {
			err = errors.Join(err, e)
			images = nil
		}
	}()
	browser, err := launchHeadless(ctx, r.chromeBin)
	if err != nil {
		return nil, err
	}
	defer func() {
		if e := browser.closeJoined(); e != nil {
			err = errors.Join(err, e)
			images = nil
		}
	}()
	for i, p := range plans {
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		if err = browser.call(ctx, browser.session, "Emulation.setDeviceMetricsOverride", map[string]any{"width": p.Width, "height": p.Height, "deviceScaleFactor": 1, "mobile": false}, nil); err != nil {
			return nil, err
		}
		if err = browser.openReady(ctx, fmt.Sprintf("%s/%d/sketch-image.html", server.base, i), "sketchImage"); err != nil {
			return nil, err
		}
		var shot struct {
			Data string `json:"data"`
		}
		if err = browser.call(ctx, browser.session, "Page.captureScreenshot", map[string]any{"format": "png", "captureBeyondViewport": false, "clip": map[string]any{"x": 0, "y": 0, "width": p.Width, "height": p.Height, "scale": 1}}, &shot); err != nil {
			return nil, err
		}
		if len(shot.Data) > base64.StdEncoding.EncodedLen(sketchimage.MaxImageBytes) {
			return nil, errors.New("sketch image: encoded PNG exceeds budget")
		}
		png, decodeErr := base64.StdEncoding.DecodeString(shot.Data)
		if decodeErr != nil {
			return nil, decodeErr
		}
		limits := []string{"replay-rendered recorded DOM; uncaptured visual content and animation phase may differ"}
		if p.Clipped {
			limits = append(limits, "ink/context clipped to the recorded source viewport")
		}
		if server.cache.hasFailures() {
			limits = append(limits, "some recorded assets were unavailable or exceeded the bounded asset budget")
		}
		images = append(images, sketchimage.Image{SketchID: p.Sketch, Plan: p, PNG: png, Limitations: limits})
	}
	if err = sketchimage.ValidateImages(plans, images); err != nil {
		return nil, err
	}
	return images, nil
}

type imageAssetCache struct {
	root   *os.Root
	mu     sync.Mutex
	bytes  int
	bodies map[string][]byte
	failed bool
}

func newImageAssetCache(dir string) (*imageAssetCache, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	return &imageAssetCache{root: root, bodies: map[string][]byte{}}, nil
}
func (c *imageAssetCache) close() error      { return c.root.Close() }
func (c *imageAssetCache) hasFailures() bool { c.mu.Lock(); defer c.mu.Unlock(); return c.failed }
func (c *imageAssetCache) get(hash string) (body []byte, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer func() {
		if err != nil {
			c.failed = true
		}
	}()
	if !recording.ValidHash(hash) {
		return nil, castassets.ErrNotFound
	}
	if body, ok := c.bodies[hash]; ok {
		return body, nil
	}
	f, err := c.root.Open(recording.AssetName(hash))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	left := sketchimage.MaxBatchImageBytes - c.bytes
	if !st.Mode().IsRegular() || st.Size() > int64(left) {
		return nil, errors.New("sketch image: asset budget/type")
	}
	body, err = io.ReadAll(io.LimitReader(f, int64(left)+1))
	if err != nil {
		return nil, err
	}
	if len(body) > left {
		return nil, errors.New("sketch image: asset byte budget")
	}
	sum := sha256.Sum256(body)
	if hex.EncodeToString(sum[:]) != hash {
		return nil, errors.New("sketch image: asset hash mismatch")
	}
	c.bodies[hash] = body
	c.bytes += len(body)
	return body, nil
}

type imageAssetStore struct {
	castassets.Served
	cache *imageAssetCache
}

func (s imageAssetStore) Get(hash string) ([]byte, error) { return s.cache.get(hash) }

type sketchImageServer struct {
	base   string
	server *http.Server
	done   chan error
	cache  *imageAssetCache
}

const sketchImageServerCleanupBudget = time.Second

func (s *sketchImageServer) close() error {
	ctx, cancel := context.WithTimeout(context.Background(), sketchImageServerCleanupBudget)
	defer cancel()
	err := s.server.Shutdown(ctx)
	if err != nil {
		err = errors.Join(fmt.Errorf("sketch image server cleanup uncertain: %w", err), s.server.Close())
	}
	serveErr := <-s.done
	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	}
	return errors.Join(err, serveErr, s.cache.close())
}
func serveSketchImages(b sketchimage.Batch, plans []sketchimage.RenderPlan) (*sketchImageServer, error) {
	token, err := broadcast.NewToken()
	if err != nil {
		return nil, err
	}
	cache, err := newImageAssetCache(b.RecordingDir)
	if err != nil {
		return nil, err
	}
	success := false
	defer func() {
		if !success {
			cache.close()
		}
	}()
	// Encode once, retaining exact line boundaries. Reject rather than creating
	// an unbounded second in-memory copy of a recording.
	var encoded bytes.Buffer
	offsets := make([]int, len(b.Records)+1)
	maxIndex := 0
	for _, v := range b.Sketches {
		maxIndex = max(maxIndex, v.Source.RecordIndex)
	}
	for i, r := range b.Records[:maxIndex+1] {
		line, e := recording.Encode(r)
		if e != nil {
			return nil, e
		}
		if len(line)+1 > sketchimage.MaxBatchImageBytes-encoded.Len() {
			return nil, errors.New("sketch image: source replay exceeds byte budget")
		}
		encoded.Write(line)
		encoded.WriteByte('\n')
		offsets[i+1] = encoded.Len()
	}
	proxies := make([]*castassets.Proxy, len(b.Sketches))
	for i, v := range b.Sketches {
		proxies[i] = castassets.NewProxy()
		store := castassets.Reopen(&recording.Recording{Dir: b.RecordingDir, Records: b.Records[:v.Source.RecordIndex+1]})
		proxies[i].Bind(imageAssetStore{Served: store, cache: cache})
	}
	prefix := "/" + token + "/r/image/"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		broadcast.SetSecurityHeaders(w.Header(), contentSecurityPolicy)
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			http.Error(w, "method not allowed", 405)
			return
		}
		path, ok := strings.CutPrefix(r.URL.Path, prefix)
		if !ok {
			http.NotFound(w, r)
			return
		}
		index, rest, ok := strings.Cut(path, "/")
		i, e := strconv.Atoi(index)
		if !ok || e != nil || i < 0 || i >= len(b.Sketches) {
			http.NotFound(w, r)
			return
		}
		v := b.Sketches[i]
		switch rest {
		case "records":
			w.Header().Set("Content-Type", "application/x-ndjson")
			w.Write(encoded.Bytes()[:offsets[v.Source.RecordIndex+1]])
		case "scene":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(struct {
				Plan   sketchimage.RenderPlan `json:"plan"`
				Source any                    `json:"source"`
				Color  string                 `json:"color"`
			}{plans[i], v.Source, v.History.Begin.Author.Color})
		default:
			if hash, yes := strings.CutPrefix(rest, "assets/"); yes {
				proxies[i].Serve(w, r, hash)
				return
			}
			serveViewerFile(w, r, rest)
		}
	})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s := &sketchImageServer{base: fmt.Sprintf("http://%s%s", listener.Addr(), strings.TrimSuffix(prefix, "/")), server: &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second}, done: make(chan error, 1), cache: cache}
	go func() { s.done <- s.server.Serve(listener) }()
	success = true
	return s, nil
}

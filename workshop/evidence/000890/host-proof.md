# BR-1: pinned host RED/GREEN evidence

Native Chrome cannot start inside the reviewer OS fence: sandbox initialization/GPU process failure occurs before the assertions. These runs use the authorized host test runner with private headless Chrome; no sandbox-disabling flag, foreground window or live fleet.

- Tested Git HEAD: `4b0b9f8e9d4d9b0c9d5c137bff879c5432319669`. `git diff --name-only HEAD -- cmd/internal/webcast` returned empty before and after both runs. Later changes in this response contain evidence/review records only.
- Host: macOS-27.0.1-arm64-arm-64bit-Mach-O. Cwd `/Users/xianxu/workspace/worktree/pair-slot9/pair`.
- Shared environment: `TMPDIR=/Volumes/TestScratch/tmp GOTMPDIR=/Volumes/TestScratch/tmp WEBCAST_LIVE_CHROME=1`.

## GREEN

```sh
go test ./cmd/internal/webcast -run '^(TestLiveViewportResizeJoinsPageHandlers|TestOperatorOwnerLiveAnnotationOff)$' -count=3 -timeout=180s -v
```

Exit0,18.909s. All three repetitions of both tests and the metrics/wait error cleanup subtests passed. [Complete log](host-green.log), SHA256 `75b436307393f70e3fa597cd20d4ff5101e280ce00ff301eee76f019369ff49a`.

## RED control

Same pinned source, with only resizeLiveViewport replaced through Go's overlay by its original metrics-only implementation:

```go
func resizeLiveViewport(call func(string, any, any) error, width, height int) error {
 return call("Emulation.setDeviceMetricsOverride", map[string]any{"width":width,"height":height,"deviceScaleFactor":1,"mobile":false}, nil)
}
```

The now-unused fmt import is removed in that overlay; the regression is unchanged. Retained overlay `/Volumes/TestScratch/tmp/pair890-red-proof.bb89m_2c/overlay.json`.

```sh
go test -overlay=/Volumes/TestScratch/tmp/pair890-red-proof.bb89m_2c/overlay.json ./cmd/internal/webcast -run '^TestLiveViewportResizeJoinsPageHandlers$' -count=1 -timeout=45s -v
```

Exit1,1.347s: `resize completed before page handlers: <nil>`. [Complete log](host-red.log), SHA256 `5a65f49a3d21c197f55b58d777bd00902943b682aed715936c2b9c57e9f10169`. This is the expected assertion failure, not Chrome startup failure. It demonstrates the existing two-phase viewport/event ordering with a held native event.

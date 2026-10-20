# #723 controlled real-stream CPU evidence

## Fixed input and scope

The first three complete minutes of pair:5’s existing wrapper scrollback are frozen locally, including startup bytes: offsets0–377464, time events20:12:47–20:15:47 PDT on2026-10-19. The capture witness matches both raw/event file identities, the prefix was reread and compared unchanged, and the sole resize is94×39. Payload377464 bytes, SHA2566b3e89fb860816fc88ad9df73129a32a5e6991764516a04a8081aea3b545d25f. Raw private content is not committed or uploaded.

Twenty independent Child owners receive the same real stream round-robin in4096-byte chunks:7549280 input bytes/run,1860 chunks/deliveries. This is one source replicated20 times, not20 independent real-client recordings. The seam is real Child.ingest/Endpoint/capture and counted queued delivery plus final input flush/Frame.Validate; it excludes compositor/presenter rendering, native terminal IO and fleet CPU. It does not estimate production utilization.

Baseline b95df08cebf86cc71cd86a27dfdf2fdb96f1b91d predates #622. Current4b3105cb2d8d4b4a99ec29e8e7d91a3974de820a is the positively identified loaded19:13 build. Production differences in the ingest/terminal/VT paths are #622’s counters and two shortcuts; the Go module also advances the Ariadne version. Both use the same Go toolchain, GOFLAGS=-p=3 -trimpath, GOMAXPROCS=3, GOGC=100, GOMEMLIMIT=off. All builds and controls precede timing. Runtime heap setup/forced GC and file reads are outside the CPU bracket; process user+system CPU covers all Go threads inside it.

## Attempt 1 — retained ingest-only diagnostic

The initial driver had a nil Sink, so publication.enqueue returned without delivery. Final capture/Frame.Validate occurred after the end CPU sample. It therefore does not measure the full intended optimized bundle. No result is discarded. Three serial observations/build used balanced order B,C,C,B,B,C; every run processed7549280 bytes and returned normalized frame/reply SHA256922de187fe9d9afe0cb8e518ceacf5575070b822acb425fd10a8c35088c9d96c.

| Run | Build | CPU seconds | Wall seconds (timed) | CPU µs/byte |
|---|---|---:|---:|---:|
| 1 | baseline | 29.293257 | 15.621156 | 3.880272 |
| 2 | current | 29.217529 | 15.489805 | 3.870240 |
| 3 | current | 29.650788 | 15.658573 | 3.927631 |
| 4 | baseline | 29.121427 | 15.430508 | 3.857511 |
| 5 | baseline | 29.296971 | 15.529761 | 3.880764 |
| 6 | current | 29.617190 | 15.656195 | 3.923181 |

Median baseline3.880272µs/byte; current3.923181µs/byte,1.105828% higher CPU/byte. This small difference does not establish a regression or improvement in the full path. Source, binary and per-run log hashes and all loads/wall times remain in the immutable local attempt-1/results.json.

## Approved correction

TLdaffacb0 explicitly authorizes one corrected six-run comparison, keeping the first attempt as ingest-only. A real Sink checks context and batch.Err and atomically counts admitted raw bytes/deliveries. Publication.Flush and Endpoint.Flush join both asynchronous lanes; final publication capture and Frame.Validate occur before the ending RUSAGE_SELF sample. Each run asserts1860 successful deliveries,7549280 delivered bytes and20 successful validations. Normalized final frame/reply hashes stay outside timing and must match across clients/builds/runs. The small positive control proves real text was rendered and counts match; empty input refuses. Both corrected binaries passed controls.

Ad hoc fresh-context review found and checked the reply-flush completion boundary and timeout retention. The corrected runner stores started-trial timeout/nonzero evidence before aborting, never retries, and requires exact count and semantic parity across all six observations. It reviewed both actual worktree heads, clean tracked sources and byte-identical driver copies; no blocking findings remained. This was an in-session measurement review, not the binary-owned close review.

Corrected results are pending at the time of this checkpoint. Invocation-private root: /Volumes/TestScratch/tmp/pair-slot5/pair622-replay.u5O9pZ. attempt-1 and ingest-only source/binaries are preserved; attempt-2-full is created exclusively and cannot be overwritten. After durable results, the owner removes the two detached worktrees; private raw evidence remains locally until acceptance.

## Reproduction source — corrected driver

Copy this exact test source into cmd/internal/ptychild/replay622_measure_test.go in each clean detached source tree, precompile with go test -c, and run the runner below. The normal test suite does not install these experiment-only files. Driver SHA256: cdf68c7ab3fe4c9a3db20fca8ea5812011b4d32e22464bbffbee04b9780ac392.

```go
package ptychild

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

type replay622Result struct {
	Deliveries        int64   `json:"deliveries"`
	DeliveredBytes    int64   `json:"delivered_bytes"`
	Validations       int     `json:"validations"`
	Bytes             int64   `json:"bytes"`
	Clients           int     `json:"clients"`
	CPUSeconds        float64 `json:"cpu_seconds"`
	WallSeconds       float64 `json:"wall_seconds"`
	CPUSecondsPerByte float64 `json:"cpu_seconds_per_byte"`
	FrameReplySHA256  string  `json:"frame_reply_sha256"`
	PayloadSHA256     string  `json:"payload_sha256"`
}

func replay622CPU(r syscall.Rusage) float64 {
	return float64(r.Utime.Sec+r.Stime.Sec) + float64(r.Utime.Usec+r.Stime.Usec)/1e6
}

func run622Replay(payload []byte, count int) (replay622Result, string, error) {
	var result replay622Result
	if len(payload) == 0 || count < 1 || count > 20 {
		return result, "", fmt.Errorf("invalid replay size or client count")
	}
	children := make([]*Child, count)
	frames := make([]terminal.Frame, count)
	var deliveries, deliveredBytes atomic.Int64
	for i := range children {
		children[i] = NewFakeChild(nil)
		defer children[i].Close()
		if err := children[i].Resize(Size{Cols: 94, Rows: 39}); err != nil {
			return result, "", err
		}
		if err := children[i].FlushOutput(context.Background()); err != nil {
			return result, "", err
		}

		children[i].SetSink(func(ctx context.Context, batch OutputBatch) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if batch.Err != nil {
				return batch.Err
			}
			deliveredBytes.Add(int64(len(batch.Raw)))
			deliveries.Add(1)
			return nil
		})
	}
	runtime.GC()
	var before, after syscall.Rusage
	start := time.Now()
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &before); err != nil {
		return result, "", err
	}
	var total int64
	for offset := 0; offset < len(payload); offset += 4096 {
		chunk := payload[offset:min(len(payload), offset+4096)]
		for _, child := range children {
			if err := child.ingest(chunk); err != nil {
				return result, "", err
			}
			total += int64(len(chunk))
		}
	}
	for i, child := range children {
		if err := child.FlushOutput(context.Background()); err != nil {
			return result, "", err
		}
		if err := child.Endpoint().Flush(context.Background()); err != nil {
			return result, "", err
		}
		pub, err := child.Endpoint().Publication(time.Now())
		if err != nil {
			return result, "", err
		}
		if err := pub.Frame.Validate(); err != nil {
			return result, "", err
		}
		frames[i] = pub.Frame
		result.Validations++
	}
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &after); err != nil {
		return result, "", err
	}
	result.WallSeconds = time.Since(start).Seconds()
	result.CPUSeconds = replay622CPU(after) - replay622CPU(before)
	result.Bytes = total
	result.Deliveries = deliveries.Load()
	result.DeliveredBytes = deliveredBytes.Load()
	result.Clients = count
	if total != int64(len(payload))*int64(count) || result.CPUSeconds < 0 || result.Deliveries != int64((len(payload)+4095)/4096)*int64(count) || result.DeliveredBytes != total || result.Validations != count {
		return result, "", fmt.Errorf("invalid accounting")
	}
	result.CPUSecondsPerByte = result.CPUSeconds / float64(total)
	payloadHash := sha256.Sum256(payload)
	result.PayloadSHA256 = hex.EncodeToString(payloadHash[:])
	var firstText string
	for i, child := range children {
		frameValue := frames[i]
		frameValue.EndpointID = "normalized"
		frame, err := json.Marshal(frameValue)
		if err != nil {
			return result, "", err
		}
		replies, err := json.Marshal(child.Writes())
		if err != nil {
			return result, "", err
		}
		h := sha256.New()
		h.Write(frame)
		h.Write(replies)
		hash := hex.EncodeToString(h.Sum(nil))
		if i == 0 {
			result.FrameReplySHA256 = hash
			var text strings.Builder
			for _, cell := range frameValue.Cells {
				text.WriteString(cell.Content)
			}
			firstText = text.String()
		} else if hash != result.FrameReplySHA256 {
			return result, "", fmt.Errorf("clients disagree")
		}
	}
	return result, firstText, nil
}

func Test622ReplayControls(t *testing.T) {
	if _, _, err := run622Replay(nil, 2); err == nil {
		t.Fatal("empty input accepted")
	}
	r, text, err := run622Replay([]byte("hello\r\nworld"), 2)
	if err != nil {
		t.Fatal(err)
	}
	if r.Bytes != 24 || r.Clients != 2 || r.Deliveries != 2 || r.DeliveredBytes != 24 || r.Validations != 2 || !strings.Contains(text, "hello") || !strings.Contains(text, "world") {
		t.Fatalf("replay control failed: bytes=%d clients=%d", r.Bytes, r.Clients)
	}
}

func Test622RecordedReplay(t *testing.T) {
	path := os.Getenv("PAIR622_REPLAY_INPUT")
	if path == "" {
		t.Skip("offline measurement input not supplied")
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	r, _, err := run622Replay(payload, 20)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("REPLAY622 %s\n", b)
}
```

## Reproduction source — corrected runner

Place alongside baseline/current directories, their compiled baseline-replay.test/current-replay.test binaries, stream.json metadata and the frozen real-three-minutes.raw. Its input hashes and exclusive attempt directory guard prevent silently changing or rerunning this experiment. Runner SHA256: f72465631f703a4712c644c01a553bf9ae5389dc4df168d9de83d39f6881e1b3.

```python
import datetime
import hashlib
import json
import os
from pathlib import Path
import statistics
import subprocess
import time

root = Path(__file__).resolve().parent
output = root / 'attempt-2-full'
output.mkdir(exist_ok=False)
stream = json.loads((root / 'stream.json').read_text())
payload = root / 'real-three-minutes.raw'
assert hashlib.sha256(payload.read_bytes()).hexdigest() == stream['sha256']
driver = root / 'replay622_measure_test.go'
driver_hash = hashlib.sha256(driver.read_bytes()).hexdigest()
for kind in ['baseline', 'current']:
    copy = root / kind / 'cmd/internal/ptychild/replay622_measure_test.go'
    assert hashlib.sha256(copy.read_bytes()).hexdigest() == driver_hash
    subprocess.run(['git', 'diff', '--quiet'], cwd=root / kind, check=True)
env = os.environ.copy()
env.update(GOMAXPROCS='3', GOGC='100', GOMEMLIMIT='off', PAIR622_REPLAY_INPUT=str(payload))
result = {'stream': stream, 'driver_sha256': driver_hash,
          'environment': {k: env[k] for k in ['GOMAXPROCS', 'GOGC', 'GOMEMLIMIT']},
          'order': ['baseline', 'current', 'current', 'baseline', 'baseline', 'current'],
          'binary_sha256': {}, 'build_commits': {}, 'scope': 'real counting publication sink; final input flush, capture and validation timed; no compositor', 'runs': []}
for kind in ['baseline', 'current']:
    result['build_commits'][kind] = subprocess.run(['git','rev-parse','HEAD'], cwd=root/kind,check=True,capture_output=True,text=True).stdout.strip()
    result['binary_sha256'][kind] = hashlib.sha256((root / f'{kind}-replay.test').read_bytes()).hexdigest()
for index, kind in enumerate(result['order'], 1):
    command = [str(root / f'{kind}-replay.test'), '-test.run', '^Test622RecordedReplay$', '-test.v']
    started = datetime.datetime.now(datetime.timezone.utc).isoformat()
    load_before = os.getloadavg()
    wall = time.monotonic()
    timed_out = False
    try:
        process = subprocess.run(command, cwd=root / kind, env=env, text=True, capture_output=True, timeout=300)
    except subprocess.TimeoutExpired as error:
        timed_out = True
        def text(value):
            return value.decode('utf-8', errors='replace') if isinstance(value, bytes) else (value or '')
        process = subprocess.CompletedProcess(command, -1, text(error.stdout), text(error.stderr))
    wall = time.monotonic() - wall
    log = output / f'{index}-{kind}.log'
    log.write_text(process.stdout + '\nSTDERR:\n' + process.stderr)
    row = {'index': index, 'kind': kind, 'command': command, 'started': started,
           'load_before': load_before, 'load_after': os.getloadavg(),
           'process_wall_seconds': wall, 'exit': process.returncode, 'timed_out': timed_out,
           'log_sha256': hashlib.sha256(log.read_bytes()).hexdigest()}
    result['runs'].append(row)
    (output / 'results.json').write_text(json.dumps(result, indent=2) + '\n')
    assert process.returncode == 0, row
    lines = [x for x in process.stdout.splitlines() if x.startswith('REPLAY622 ')]
    assert len(lines) == 1, process.stdout
    measured = json.loads(lines[0].removeprefix('REPLAY622 '))
    row['measurement'] = measured
    assert measured['bytes'] == stream['total_replay_bytes']
    assert measured['clients'] == stream['clients']
    assert measured['delivered_bytes'] == stream['total_replay_bytes']
    assert measured['deliveries'] == ((stream['bytes'] + 4095) // 4096) * stream['clients']
    assert measured['validations'] == stream['clients']
    assert measured['payload_sha256'] == stream['sha256']
    (output / 'results.json').write_text(json.dumps(result, indent=2) + '\n')
    print(json.dumps({'index': index, 'kind': kind, **measured}), flush=True)
assert len({x['measurement']['frame_reply_sha256'] for x in result['runs']}) == 1
assert len({(x['measurement']['deliveries'],x['measurement']['validations'],x['measurement']['delivered_bytes']) for x in result['runs']}) == 1
medians = {kind: statistics.median(x['measurement']['cpu_seconds_per_byte'] for x in result['runs'] if x['kind'] == kind)
           for kind in ['baseline', 'current']}
result['median_cpu_seconds_per_byte'] = medians
result['reduction_percent'] = 100 * (1 - medians['current'] / medians['baseline'])
result['improved'] = medians['current'] < medians['baseline']
(output / 'results.json').write_text(json.dumps(result, indent=2) + '\n')
print(json.dumps({'medians': medians, 'reduction_percent': result['reduction_percent'], 'improved': result['improved']}), flush=True)
```

## Corrected result — 2026-10-19

All six corrected observations completed without retries. Every run delivered1860 batches/7549280 bytes and completed20 timed final frame validations; final frame/reply hashes match the first attempt and each other. Publication and reply queues were joined inside timing. Controls passed for both builds; a deliberate missing-Sink control variant failed with invalid accounting, verifying that the earlier missing stage is now detected. The original driver was restored after this control; measured binaries/results were untouched.

| Run | Build | CPU seconds | Wall seconds (timed) | CPU µs/byte | Deliveries | Validations |
|---|---|---:|---:|---:|---:|---:|
| 1 | baseline | 29.148599 | 28.024173 | 3.861110 | 1860 | 20 |
| 2 | current | 28.804248 | 27.106092 | 3.815496 | 1860 | 20 |
| 3 | current | 29.291803 | 28.204541 | 3.880079 | 1860 | 20 |
| 4 | baseline | 29.479283 | 27.506442 | 3.904913 | 1860 | 20 |
| 5 | baseline | 30.357654 | 30.665584 | 4.021265 | 1860 | 20 |
| 6 | current | 30.432552 | 33.027379 | 4.031186 | 1860 | 20 |

Median CPU/byte: baseline3.904913µs; current3.880079µs, an observed0.635972% reduction. Baseline range3.861110–4.021265µs/byte; current3.815496–4.031186µs/byte. The ranges overlap: three observations and this one stream establish a small lower sample median, not a robust general speedup or a forecast of live fleet CPU. Contrast the incomplete ingest-only median at+1.105828% CPU/byte. Both outcomes are reported; no unfavorable observation is omitted.

The final validation runs once per client, not once per4096-byte batch, and the counted Sink acknowledges output without rendering a console. The experiment measures exactly that fixed work. Earlier synthetic ASCII/mixed redraw results remain separate evidence and are not substituted for this real-stream comparison.

Local corrected results SHA256: 2358b4ba182ef759f48770017a09c99dd9a9c7c138518580698bae5aa8b24e15. Per-run log hashes (in order):

- 1 baseline: ba36a689007dcb9c184c9b9196c3bc573b2ee8c76fb1cbed01bb71c113ad263e
- 2 current: 655c433b1317474030dae82743401763eceb3779ae4e5e72c7a19e2a387b89a3
- 3 current: f9f80215c23b965fb90ddb4a24bbcc35d272ed89fd076d6ba650060bc3a6786a
- 4 baseline: 550f369fcbf43006e6b759cd10219d9e69de6bddac7099fdfd6256d5bd580510
- 5 baseline: 74946ee1d9ee4b412e67f4d5a28c5d78b070882a5508161d4c36e63a9fdbcc90
- 6 current: db04f8fd838693dd090aa7837bf589636d25ff7dadd079a6604f359e7246066c

The result file also retains every load triplet, process wall duration, CPU bracket values, binary/driver/stream hashes and Git heads. No raw terminal payload is present in the committed report.

Both owned detached worktrees were removed after all runs/controls, with tracked source clean and only the known injected driver untracked. Source/binaries/raw and both attempt directories remain in the invocation root; no other worktree or process was touched.

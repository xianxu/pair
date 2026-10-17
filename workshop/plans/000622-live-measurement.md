# #622 live measurement, 2026-10-16 22:25–22:29 PDT

TL dispatch 4306f6e3 authorizes this measurement and the M2 proposal, not implementation. Raw evidence is retained locally at /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair622-m2.kftNtp. Native samples are interval-a-server.sample.txt (22:25:33.873) and interval-b-server.sample.txt (22:27:15.008), five seconds each, sample PID99807.

## Identity and limitations

Live resources identify server PID99807, package start 2026-10-16T22:22:47.640652-07:00; ps/sample identify OS launch22:22:40.062. TL attests startup build712a6478515d6620d2f453856592af65f2ce32a3. Running executable inode749561659 differs from current disk inode749587249; the latter's clean build metadata does not establish the loaded revision. This pre-existing rebuild ambiguity belongs to #624. Counters are positively supported by the live server.

Both intervals have zero omitted clients. A has23→22 clients (brain:1 endpoint16 disappears); B has22→23 (endpoint25, PID41556/generation1792214787.144472 appears). Compare only matching server identity, endpoint, thread and process PID/generation. No final total is available for endpoint16 and no start for25: neither is zero. Whole-server CPU includes changing membership and cannot be allocated by bytes. These are natural mixed-load observations, not controlled idle-fleet or per-client CPU measurements.

| Interval | Counter timestamps PDT | Counter duration | Matched bytes (22 clients) | CPU brackets | CPU elapsed (monotonic) | CPU seconds | One-core equivalent |
|---|---|---:|---:|---|---:|---:|---:|
| A |22:25:15.123187–22:26:30.476910|75.353723s|121772431|1:44.16→2:33.46|76.054908s|49.30|64.82%|
| B |22:26:30.476910–22:28:43.500349|133.023439s|146010797|2:33.46→3:18.65|132.910273s|45.19|34.00%|

CPU and resource reads are adjacent, not atomic: timestamp skew is retained. RSS546816→550336→550448KiB. Load averages at A start61.49/53.36/39.93, A end51.41/53.00/41.00, B end51.57/54.06/43.30. Several full-suite/merge processes ran concurrently. Observer environment GOMAXPROCS=3, GOFLAGS=-p=3; running-server effective values were not established. Console selection was not observable from the diagnostic response and was not changed. No live input probes were sent (#629 not landed).

Read-only live wrapper peeks establish ariadne:0 and parley.nvim:0 remained settled, empty composer, unchanged22:23 completion footers throughout the observed window. Their rates are idle-client output, not machine-idle rates. pair:5 is this observer's active turn. pair:3 reports an open turn waiting on a background suite; ops:0 is actively working during A and settles during B. Do not classify all non-idle clients as CPU-busy. Initial broad A-start peek returned a running shell response that was mistakenly treated as empty output; interval-a-peek-start.json is invalid and excluded. Earlier six-client live peek, mid and end peeks supply the positive idle evidence; failure is preserved.

## Matched per-client output

Decimal bytes/second, rounded. A/B durations are above.

| Client | Endpoint | A delta bytes | A B/s | B delta bytes | B B/s |
|---|---|---:|---:|---:|---:|
| ops:0 | couch-pty-1 | 10208021 | 135469 | 12229501 | 91935 |
| pair:7 | couch-pty-3 | 5097408 | 67647 | 6167641 | 46365 |
| pair:6 | couch-pty-2 | 5551923 | 73679 | 6457978 | 48547 |
| ariadne:5 | couch-pty-4 | 5136986 | 68172 | 6303840 | 47389 |
| pair:3 | couch-pty-5 | 5127115 | 68041 | 6106003 | 45902 |
| parley.nvim:1 | couch-pty-7 | 5022493 | 66653 | 6109545 | 45928 |
| pair:1 | couch-pty-6 | 5149793 | 68342 | 6316374 | 47483 |
| ariadne:3 | couch-pty-8 | 5177289 | 68707 | 6432812 | 48358 |
| ariadne:4 | couch-pty-9 | 4909496 | 65153 | 5858974 | 44044 |
| pair:2 | couch-pty-10 | 5295154 | 70271 | 6327708 | 47568 |
| parley.nvim:4 | couch-pty-11 | 5108832 | 67799 | 5971968 | 44894 |
| parley.nvim:3 | couch-pty-12 | 4857930 | 64469 | 5621319 | 42258 |
| pair:5 | couch-pty-13 | 5372579 | 71299 | 6868663 | 51635 |
| pair:4 | couch-pty-14 | 4739248 | 62894 | 5276226 | 39664 |
| parley.nvim:2 | couch-pty-15 | 5375009 | 71331 | 6377526 | 47943 |
| ariadne:0 | couch-pty-17 | 5321592 | 70622 | 6320952 | 47517 |
| ariadne:2 | couch-pty-18 | 8128430 | 107871 | 10482035 | 78798 |
| xianxu.dev:0 | couch-pty-19 | 5826884 | 77328 | 6680283 | 50219 |
| pair:0 | couch-pty-20 | 5118813 | 67931 | 5991606 | 45042 |
| ariadne:1 | couch-pty-21 | 5432239 | 72091 | 6545352 | 49204 |
| brain:0 | couch-pty-22 | 5149914 | 68344 | 6082246 | 45723 |
| parley.nvim:0 | couch-pty-23 | 4665283 | 61912 | 5482245 | 41212 |

Matched aggregate: A1.616MB/s; B1.098MB/s. Confirmed idle ariadne:0 and parley.nvim:0 span41–71kB/s, overlapping many working clients. This supports investigating always-present hosted output; it does not prove identical redraws, redundant bytes or which inner producer emitted them. Hosted Pair resumes into a blocking Zellij attachment, so ingress covers the attached session, not merely agent text. Title polling already suppresses unchanged pane names.

## Native profile interpretation

Sample A call-graph symbol occurrence weights: Child.ingest76, handleGrapheme60, Parser.advance60, Emulator.Write56, Endpoint.Feed47, Endpoint.capture43, handlePrint40, Presenter.paintEndpoint34. These are inclusive overlapping weights, not CPU percentages or additive subsystem shares. Separate leaf summary includes handleGrapheme43, grapheme Iterator.Next31/First15 and FirstGraphemeCluster13; presentation includes capture35, rowMetadata29, plainText29 and validateCell25. JSON and GC/runtime work also matter. Sample B is retained as a second natural observation; no unsupported quantitative allocation is inferred from it.

## Decision comparison

| Candidate | Consequence | Recommendation |
|---|---|---|
| Visible-only emulator + lossy128KiB raw ring | Cannot reconstruct earlier screen/modes/history; hidden queries/effects still need timely parsing | Reject; existing ring-eviction regression proves lost authority |
| Exact checkpoint + bounded lazy replay | Every byte still needs parsing; continuous idle output triggers catch-up; query/effect/resize/EOF chronology requires eager parsing or a broader backend redesign | Defer; substantially larger mechanism without demonstrated net saving |
| Narrow eager ASCII shortcuts | Eliminates known general grapheme scans while preserving the single authority, ordering and hidden consumers | Proposed M2, contingent on differential correctness and measured complete-ingest benefit |
| Reduce upstream redraw output | Could save more parsing and transport | Future evidence-dependent option; current byte counters cannot identify redundant frames |

Proposed source mechanisms: third_party/vt/utf8.go handlePrint segments an ordinary adjacent ASCII pair, then segments the new single ASCII rune again. A shortcut requires BOTH existing cluster and next rune to be single printable ASCII; preserve provisional state and handleGrapheme so later combining/keycap extensions, charsets, wrapping and cursor remain correct. terminal/frame.go validateCell also segments each ordinary ASCII width1 cell; skip that scan only after retaining safety/link/width checks. No new parser, worker, mode, visibility policy or retained raw buffer.

Raw manifest SHA256: 7cc0850c49d4eb1ddae8a48fa71a63afb13f322a4dfda692d4d76cd02c893283. Profile A SHA256: 7bceec43f581b2e0ee56f15f86a9ca9abf89fbe436dbc476acc9eb5246b91fc0; profile B: 767ab3762708a0bb1389005c26ba89c1ff1bdd0639e7601ac645fe02d6575c73. Sample B independently includes grapheme segmentation and validateCell; native waiting/syscall stacks also dominate much of its call graph, so they cannot be counted as busy CPU.

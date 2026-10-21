# #755 browser qualification evidence

Committed code head: 8fefb5c78182993499354d9227052209570022ea.

Run on the worker macOS host, with the default installed Google Chrome, a private temporary profile and headless mode. Chrome sandboxing was enabled; no sandbox-disabling argument was used. This records host qualification for BR-4, whose review environment could not initialize Chrome. The following evidence commit changes documentation only.

Command:

```sh
env -u COUCH_WEBCAST_CHROME WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/mockserve -run '^(TestMockBrowserReload|TestSampleBrowser)$' -count=1 -v
```

Returned output (exit 0):

```text
=== RUN   TestMockBrowserReload
=== RUN   TestMockBrowserReload/edits_navigation_and_repair
=== RUN   TestMockBrowserReload/change_before_first_poll/switch=false
=== RUN   TestMockBrowserReload/change_before_first_poll/switch=true
--- PASS: TestMockBrowserReload (6.82s)
    --- PASS: TestMockBrowserReload/edits_navigation_and_repair (4.84s)
    --- PASS: TestMockBrowserReload/change_before_first_poll/switch=false (0.99s)
    --- PASS: TestMockBrowserReload/change_before_first_poll/switch=true (1.00s)
=== RUN   TestSampleBrowser
--- PASS: TestSampleBrowser (0.68s)
PASS
ok  github.com/xianxu/pair/cmd/internal/mockserve 7.686s
```

The package race run also passed after the BR-2/BR-3 fixes: `go test -race ./cmd/internal/mockserve -count=1` (1.910s). It includes the deterministic client lifecycle regressions through Node and the concurrent selection identity tests. Browser tests cover reload, navigation, shared fixtures, repair, the first-poll race and the actual sample. The JavaScript lifecycle tests separately cover repeated BFCache restore and delayed callbacks.

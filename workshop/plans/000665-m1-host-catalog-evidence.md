# #665 M1 exact-head host catalog evidence

BR-4 permits measured host evidence for the exact reviewed source against identified main authority. The disposable review checkout had no main authority and correctly refused; the registered host checkout has the following real configuration and refs. No ref/configuration was fabricated or changed for this measurement.

- Host checkout: /Users/xianxu/workspace/worktree/pair-slot1/pair
- Measured HEAD: 6d7314450a44b1a84a84605c691678390398e10e
- branch.main.remote: origin
- branch.main.merge: refs/heads/main
- refs/remotes/origin/main: a5bc8e64a82776647060495425ec311dee235b23
- Merge base of HEAD and that authority: d8ccc4577351b3dde5f829e83f5d83e4fedced5d
- git diff --exit-code HEAD -- '*.go' go.mod go.sum: exit0 (source/module inputs match the measured commit).

Executed command, exit0:

```text
go test ./internal/repolint -run '^TestTrackedGoFilesPassCatalog$' -count=1 -v
```

Returned output:

```text
=== RUN   TestTrackedGoFilesPassCatalog
    catalog_test.go:460: checked 2559 tracked Go sources, 5 rules; 100 exact sleep debt sites
--- PASS: TestTrackedGoFilesPassCatalog (2.52s)
PASS
ok  	github.com/xianxu/pair/internal/repolint	2.659s
```

Retained log: /tmp/pair665-m1-r8-exact-head-catalog-green.log

SHA256: e6bf6bc27dd850a817c7a59b48e318df7478ffe6f74d3ecce3935b7b34edd694

This is targeted catalog evidence, not a recorded full-suite PASS. The catalog's missing/ambiguous-authority refusal is unchanged. The host facts above were returned by git rev-parse, git config, git show-ref and git merge-base before this evidence-only commit; subsequent review bookkeeping changes no Go or module source.

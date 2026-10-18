---
id: 000663
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '413dfa993fff27cd81fe0f02f983721f6eed66a8' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T02:55:47-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# Attribute remaining final-Go suite device writes after fixture and cache reductions

## Problem

After pair#658’s fixture/cache reductions, the same complete Make suite still wrote15,098,630,144 physical-device bytes in643.772099s. The largest remaining measured interval is final go test ./... through completion:13,312,307,200 bytes (about88.2% of the device total). Its producer remains unproved. Sampled own-process counters across the entire suite total only1,038,913,536 bytes/825 generations, leaving a14,059,716,608-byte difference that includes missed short-lived processes, filesystem work and other host activity. It must not be assigned wholesale to tests, Git or APFS metadata.

## Spec

Attribute the remaining final-Go device interval with a bounded measurement before proposing another optimization. Distinguish brief compiler/link/Git/test descendants, metadata/durability effects and other host writers. Existing APFS metadata counters are non-additive clues, not a causal partition. Prefer a small owned-process probe; if unprivileged evidence cannot name the producer, ask the hub for an operator-run time-bounded60s fs_usage command filtered to the test processes. Do not loop full suites or change durability/coverage on a guess.

## Done when

- Retained measurements name the dominant remaining producer/path or explicitly demonstrate the attribution limit and required operator trace.
- Any proposed reduction targets that measured producer while retaining tests and durability; a counted byte/file budget defends it.
- Before/after device comparisons state body duration, background windows, sampled-process gaps and non-additive APFS scope.

## Plan

- [ ] Obtain hub approval for workshop/plans/000663-device-attribution-plan.md before measurement code or new runs.
- [ ] Compare observation modes on identical completed couchcore work, then use a bounded native host census if needed; retain coverage limits and raw counter endpoints.
- [ ] Assess one versus two active suites using bytes per equal completed work; obtain approval before any full-suite comparison.
- [ ] Report the measured producer or the precise operator trace needed; revise the fix plan before implementation.

## Revisions

- 2026-10-18: Hub approved only the first observation comparison, one attempt (receipt f22cead8-9b80-4ee6-a1e4-8e4d9b2e7a3d). Cancel the host census. If attribution remains open, supply a scoped 60-second fs_usage command for the operator; defer concurrency until the writer is known. Fixed work is five invocations per arm, full observer first, endpoints only second, at most 60 seconds each.

- 2026-10-18: Hub assigned attribution first and added the machine concurrency question. Both existing complete windows already ran one Make holding both tokens. No production fix or global concurrency change is selected. Start with a bounded workload observation control; reserve a full equal-completion concurrency comparison for a separate approval.

## Log

2026-10-18 pre-run review: the first admission wait was cancelled before acquiring tokens or starting either arm, to correct temporary instrumentation. Review found that an in-loop sampler could overrun the arm bound and the ending counter preceded owned-group cleanup. Added five-second ioreg/ps limits, an independent 60-second arm watchdog with bounded shutdown, and cleanup before the ending counter. Only our identified queue helper was interrupted; its parent exited and the cancelled queue record is retained. This loses our queue position but is not a measurement retry: the one allowed comparison is still unrun. No host census or concurrency experiment will be built under the current approval.

2026-10-18 planning: claimed and ran start-plan on branch 000663-remaining-suite-device-writes. Read-only analysis of retained evidence found final-Go rates of 43.392 MB/s before and 40.851 MB/s after, over 358.452 s and 325.877 s. Tail rates fell to 3.222/3.409 MB/s while sampling continued at about 1.46 Hz and the retained PID set grew. This contradicts a simple fixed write cost per observer tick but does not exclude workload-dependent observer effects. NativeResourceReader already provides bounded enumeration and identity-fenced disk counters; reuse that seam (ARCH-DRY). Existing process totals miss short-lived processes and exclude the observer and unrelated host processes. No new measurement or implementation has run for this issue.

From pair#658’s approved unchanged #662-backed rerun, source030bdf8a41810d07df15ba68bd5d8295c056b22f, clean at Make start. Original116 final Go packages all pass before; after118 all pass, adding testdisk/testgit. Make/supervisor exit0, all direct children reaped, both ordinary suite tokens held through30s before/after windows. The first incomplete resolver-failure attempt is retained separately in #662.

| Same Make body | Before | After |
|---|---:|---:|
| Physical disk0 bytes |19,768,967,168|15,098,630,144|
| Body seconds |844.188058|643.772099|
| Average MB/s |23.417729|23.453378|
| Before final Go interval |4,214,910,976|1,786,322,944|
| Final Go interval |15,554,056,192|13,312,307,200|

The observed4,670,337,024-byte reduction (23.625%) accompanies a200.416s shorter body, near-identical write rate and different quiet slopes; it is not an isolated payload-saving measurement. Quiet-before/after slopes:3.943917/2.481531MB/s before;2.882504/0.889555 after. Registry4294970125 is stable; all976 after samples match embedded data, no counter decreases, exact endpoint subtraction agrees. Final-Go boundary uses the nearest sample to the recipe event (+0.201017s before/+0.202169s after); do not mix it with the distinct later-sample phase-label bucket.

After APFS nearest windows have−11.024661s/+6.077542s endpoint errors. Data volume:2,733,666,304 block bytes /4,515,151,872 metadata bytes; AgentScratch:1,669,398,528 /84,103,168. These counters overlap and cannot be added to device totals. Swapouts/Pageouts unchanged; no NAND-wear claim.

Durable evidence is pair#658’s workshop/plans/000658-device-write-baseline.md (archived with that issue when landed). Local raw after root:/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair658-662-device-attempt2.DViNQs/device-suite/{result.json,device.jsonl,phases.jsonl,volumes.jsonl,make.log,analysis.json,counter-checks.json,body-source.json,supervisor-exit.json}. Baseline root:/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair658-profile.yqKPI1/device-suite. Raw temporary files are not the durable record.

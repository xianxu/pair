---
type: continuation
slug: drive-month-upload-wiring
agent: codex
created: 2026-10-19T21:04:35
branch: 000706-drive-backup-store-bundles-in-year-month-folders-instead-of-one-flat-folder
worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
issues: [000706]
---

# Continuation: drive-month-upload-wiring

## NEXT ACTION

Implement the outer month-aware upload coordinator around the existing per-folder verification in `cmd/internal/historyarchive/drive.go`. First read #706's issue and durable approved plan. Run the three tests in `drive_months_external_test.go`: two are intentionally RED against the unchanged flat writer (month folder absent and partial legacy pair repaired), while full legacy adoption passes. Preserve final-list verification and the single global 4 GiB cap while making those tests GREEN. M1 is not complete.

## State of play

pair#706 is owned by pair:6, branch `000706-drive-backup-store-bundles-in-year-month-folders-instead-of-one-flat-folder`. Hub approval f445934d-d16f-41e0-b552-46ff4ce15307 is current; change-code passed CLEAN after PQ-1 function-specific testing correction and adopted estimate 4.73 h. No review boundary closed, PR or full suite begun, and no real Drive mutation performed. Source checkpoint commit 2423126d1 contains completed components and expected behavioral RED tests. Run sdlc state after restart.

#689 landed PR450, df16e5f535d3fabc46044958030c8f1e1280f007; follow-up #719 (periodic operator-present restore drill) is filed unclaimed. #707 is parked. #710 owns gap-5 scheduling/early collection failure, new kinds/Couch snapshots/catalog encryption/bootstrap; do not expand #706 into it.

## Thread arc & user model

The operator wants recoverable encrypted history with measurable bounded Drive cost. #678 provided flat batched uploads, #689 recovery/index, and #706 now bounds growing inventories by UTC year/month without moving old backups. The hub approves designs and arranges real Drive checks; proceed through gates without unnecessary prompts. Report when mixed legacy/new upload and restore qualification is ready, before landing. No operator session or real remote work by the worker.

## Artifact map

Read `workshop/issues/000706-drive-backup-store-bundles-in-year-month-folders-instead-of-one-flat-folder.md`, then `workshop/plans/000706-drive-month-folders-plan.md`. The plan carries approved invariants and named risky-function test strategies. The issue Log records gates, expected RED evidence and checkpoint.

Completed tested production seams: `drive_layout.go` uses label.From UTC and accepts only exact configured root or canonical month; `drive_folder_model.go` is the explicit event/effect state model; `rclone_folders.go` implements ResolveChild(ctx,parent,name,cachedInventory) and EnsureChild with one mkdir and positive pinned observation; `drive_receipt_json.go` preserves unknown receipt and nested-object fields with bounds and strict known fields. MatchReceipt compares known fields explicitly, ignoring opaque extensions and verification time. New source paths registered in artifactpath manifest.

`archivetest/rclone.go` now supports hierarchical folder IDs: FolderObjects and ChildFolders opt into hierarchy (set nonnil even empty); legacy Objects remains root and Objects-only fixtures retain old arbitrary-pin compatibility. BeforeFolder and FolderLists provide fault injection/counts; mkdir loss-of-ack retains effect. Agents completed, no agent work remains outstanding. Their files are committed.

Fresh focused command PASS: go test ./cmd/internal/historyarchive ./cmd/internal/historyarchive/archivetest -run 'TestDriveMonthLayout|TestDriveFolderSequences|FuzzDriveMonthLocation|FuzzDriveFolderEvents|TestUploadReceipt|TestUploadObjectJSON|TestMatchReceipt|TestDriveJSON|TestDriveDestination|FuzzUploadReceiptJSON|TestRclone|TestDecodeFlatInventory|TestFakeRclone' -count=1. Broader fake race and receipt fuzz smoke passed earlier. These exclude the two deliberate upload RED tests; do not claim M1 or package green.

## Live deliberations

Existing DriveMirror.Run is still the original single-folder implementation. Suggested extraction: outer coordinator projects catalog once, validates root-namespace receipts without rescanning recorded historical months, lists legacy root, groups pending labels by month, refuses any partial/conflicting root pair, adopts complete root pairs, resolves cached year/month IDs and delegates existing batch verification per physical folder. Carry consumed bytes across groups; a zero remaining budget must not become the default cap. Only current and pending months get listed; root and needed year inventories are cached, unrelated history stays untouched. Folder failures should retain truthful pending counts while local backup remains independent.

Current report has one Inventory pointer. M2 needs inventories per physical folder plus compatible reporting, not a fabricated tree count. Receipt storage keys stay configured-root namespace but Destination and FolderID identify the actual leaf. The original verifier's receipt-key calculation currently uses physical destination namespace; change that deliberately in extraction.

M2 then updates label_index_view receipt projection, DriveIndexSync bounded two-level labels-only atomic snapshots, restore plan recorded locations and grouped pinned acquisition, Couch warning counts per folder, docs and mixed-layout round trip. Existing index complete-observation precedence from #689 must be reused, not replaced with another clock. Restore always full ciphertext SHA-256 before decode. Operator's real Drive fresh upload/retry plus old/new restore is required before landing.

## Decisions & dead ends

No automatic migration, recursive listing per upload, deletion, repair of ambiguous folder/object names, or weakening uniqueness checks. Month comes from label.From UTC, matching BundleKeys. Tests with nonzero timezone were invalid fixtures; convert fixture to UTC rather than weakening ValidateLabel.

Parked stash cf709527564ea17b03504c0fb767c56a76d70504 contains only generated .github/workflows/local-gates-guard.yml and merge-check.yml pending CI migration. Hub expressly says leave it parked; no workflow edits or operator verification labels enter #706.

## Lessons learned

PQ-1 required each risky function's adversarial class and mechanical guard rather than long prose test inventories. This is recorded in workshop/lessons.md. Bounded generated state sequences, fuzz seeds and process/count invariants complement the stateful fake.

Use bare sdlc start-plan only; every other command uses /Users/xianxu/workspace/ariadne/bin/sdlc until the stale sibling binary issue is resolved. Couch whole command, sends via --send-to hub --message TEXT; check submitted receipt. Agent trailer: codex/gpt-6.1-sol/medium. Binary milestone-close/close dispatch mandatory fresh review; don't duplicate review. Full suite is recorded on pushed head through local-check, and M1/M2 boundaries remain open. Keep queue runners unchanged and ask hub for decisions, not an in-pane prompt.

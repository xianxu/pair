# Agent scratch volume

Agents generate file churn at a rate macOS's file-change machinery is not tuned for: Go's build cache (one cache key per
worktree path, so every slot adds its own copy) and the temp roots tests create and delete by the thousand. fseventsd
journals every change on the Data volume and delivers it to watchers. Spotlight Privacy exclusions do not help, because
they filter after fseventsd has already journaled and delivered the event. Under a fleet of slots fseventsd grew past
10 GB and ran above 100% CPU (pair#496).

The fix is configuration, not pair code. The churn goes to a separate APFS volume that fseventsd does not journal,
Spotlight does not index and Time Machine skips. Each tool is pointed at it through its own per-user setting, never the
shell profile, so the setting covers Couch slots, hand-started pair sessions and the operator's terminal alike.

## Layout

| Path | Used by | Set by |
|---|---|---|
| `/Volumes/AgentScratch/go-build` | Go build cache | `go env -w GOCACHE=…` (Go's own `$(go env GOENV)` file) |
| `/Volumes/AgentScratch/tmp` | Go build temp files | `go env -w GOTMPDIR=…` |
| `/Volumes/AgentScratch/tmp/claude-<uid>` | Claude Code's per-command sandbox `TMPDIR`, which `t.TempDir()` and `mktemp "${TMPDIR}"` test roots follow | `env.CLAUDE_CODE_TMPDIR` in `~/.claude/settings.json` |

Claude Code replaces `TMPDIR` for every sandboxed command, so an inherited `TMPDIR` would be ignored there.
`CLAUDE_CODE_TMPDIR` is the setting that moves that directory, and both it and `sandbox.filesystem.allowWrite`
take effect in running sessions without a restart. Another agent needs its own equivalent of both settings: its temp
directory and its sandbox's writable roots.

## Setup

Every step changes the machine outside the repository, so each one needs the operator's approval first. The `sudo`
steps need a real terminal, because the `!` prefix gives sudo no tty to read the password from.

```sh
diskutil apfs addVolume disk3 APFS AgentScratch -quota 200g   # a volume in the existing container: no partitioning, no resize
sudo mkdir -p /Volumes/AgentScratch/.fseventsd
sudo touch /Volumes/AgentScratch/.fseventsd/no_log            # no fseventsd journal for this volume
diskutil unmount /Volumes/AgentScratch && diskutil mount AgentScratch   # fseventsd reads no_log at mount
sudo mdutil -i off /Volumes/AgentScratch                      # no Spotlight index
sudo tmutil addexclusion -v /Volumes/AgentScratch             # -v for a volume; -p fails with "Invalid argument"
mkdir -p /Volumes/AgentScratch/go-build /Volumes/AgentScratch/tmp
go env -w GOCACHE=/Volumes/AgentScratch/go-build GOTMPDIR=/Volumes/AgentScratch/tmp
```

`~/.claude/settings.json`, merged into the existing keys:

```json
"env": { "CLAUDE_CODE_TMPDIR": "/Volumes/AgentScratch/tmp" },
"sandbox": { "filesystem": { "allowWrite": ["/Volumes/AgentScratch"] } }
```

The container (`disk3` here; `diskutil apfs list` names it) is shared space. The quota is a ceiling, not a reservation:
unused space stays with the other volumes.

## Verify

- `mdutil -s /Volumes/AgentScratch` reports `Indexing and searching disabled`.
- `sudo tmutil isexcluded /Volumes/AgentScratch` reports `[Excluded]`.
- `sudo ls -la /Volumes/AgentScratch/.fseventsd` lists only `fseventsd-uuid` and `no_log`, even after the cache has
  grown. A journal page appearing there means `no_log` was not read at mount.
- In an agent session: `go env GOCACHE` and `echo $TMPDIR` both point into `/Volumes/AgentScratch`.

## Lifecycle

- Go trims its cache itself, dropping entries unused for about five days.
- Nothing sweeps leaked temp directories. The 200 GB quota bounds them; when the volume fills, builds and tests fail
  with ENOSPC, and nothing outside the volume is affected. Delete the contents of `tmp/` (or recreate the volume) to
  recover.
- The volume holds nothing that needs keeping, so it is never backed up or migrated.

## Teardown

```sh
go env -u GOCACHE GOTMPDIR                     # Go falls back to ~/Library/Caches/go-build
# remove env.CLAUDE_CODE_TMPDIR and the /Volumes/AgentScratch allowWrite entry from ~/.claude/settings.json
diskutil info AgentScratch                     # confirm the target before deleting
diskutil apfs deleteVolume AgentScratch        # its space returns to the container
```

If the volume ever fails to mount while Go still points at it, Go fails loudly because it cannot create the cache
directory. `go env -u` is the immediate fix.

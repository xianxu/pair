# #767 native descriptor probe

macOS /bin/sh: GNU bash3.2.57 arm64-apple-darwin26. Eight workers use Go exec
ExtraFiles to pass a parent-owned regular0600 lease at FD3. Each child is a shell
script with separate stdout/stderr forwarding. No child or parent closes the
lease before the shell exits.

Old gate: `test -r /dev/fd/3`. On its failure the probe executes builtin read from
FD3 before launching any diagnostic child. Worker0 sample968 failed; the other
seven workers completed2000 each: one failure in14969 starts. Exact diagnostic:

```
worker0 sample968: exit status 42: READABLE_GATE_FALSE token=lease-token read_rc=0
mode=100600 uid=501 inode=58928009
```

The descriptor was readable and delivered its exact token after the pathname
probe rejected it. The following stat's attributes were normal; the original
predicate does not preserve its underlying errno. This proves an unsound oracle,
not the exact internal XNU interleaving. No production FD-lifetime change is
justified by this evidence.

The corrected probe source is `fd-probe.go.txt`: direct builtin read plus exact
token comparison. All eight workers completed2000, 16000 successful starts.
To repeat, copy the source to main.go inside a checked mktemp directory under
TMPDIR (prefer TestScratch when mounted), then run `go run main.go DIR` with that
same private directory. Each child is foreground-owned and joined; the files
remain only under DIR until explicitly removed. Append `old` to the probe arguments to repeat the original predicate with its
failure diagnostics. The probe counts every attempt and returns nonzero on any
child failure. The committed identity test independently provides a
deterministic mode000 regression and absent/wrong-token negative controls.

Primary-source context:
- https://github.com/apple-oss-distributions/bash/blob/main/bash-3.2/test.c
- https://github.com/apple-oss-distributions/bash/blob/main/bash-3.2/lib/sh/eaccess.c
- https://github.com/apple-oss-distributions/xnu/blob/main/bsd/miscfs/devfs/devfs_fdesc_support.c

Bash's /dev/fd permission predicate uses pathname stat/access attributes; XNU
implements that through fdesc vnode lookup/getattr. The precise reason that
observation transiently failed was not captured and is not required for replacing
it with the actual inherited-resource operation. No timeout was increased.

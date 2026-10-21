## Zellij 0.45.1 false session misses
Requires Bash, stock Unix utilities, and `zellij 0.45.1`; no Pair or Couch.
Run `bash /path/to/zellij-false-miss.sh 1000 4`; it needs no checkout.
Arguments are total raw queries and parallel workers (defaults: 1000 and 4).
Three background sessions use fresh HOME, XDG, config, and socket roots; use a short TMPDIR if the socket-path check refuses.
Queries mix list-panes JSON, list-clients, and unformatted list-sessions.
`exit0_target_listed` counts target-not-found stderr with that target in stdout.
`exit1_alive_confirmed` requires a socket and subsequent layout response; inconclusive cases stay separate.
Raw stderr-bearing results and totals print before private-session cleanup; redirect stdout to retain them.
Zero observed misses proves no absence of the race; cleanup uncertainty retains its named root and fails.

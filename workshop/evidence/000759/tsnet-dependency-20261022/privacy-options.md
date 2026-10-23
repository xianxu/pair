# Log upload prevention at v1.104.2 (static review only)

No tsnet.Server DisableLogTail or NoLogsNoSupport field exists in tsnet/tsnet.go:220-345. Start always calls startLogger (:874); startLogger (:1091-1127) instantiates logtail and filch. Logf/UserLogf suppressions do not disable this path.

Public no-environment mechanism: tailscale.com/logtail.Disable(), logtail/logtail.go:735-737, sets an atomic process-wide irreversible flag for lifetime of process. New sendLocked calls drop entries before buffering (:766-770); upload loop checks flag before upload (:531-535). Call BEFORE any Start/LocalClient/Listen so no previously in-flight upload exists. It affects every logtail instance in the process, not just this node; not a server-scoped API. It does not stop state/config/log-file setup, disable other network traffic/metrics, or assert no telemetry elsewhere. No code has been called in this review.

Logger.SetEnabled(false) exists (:740-746) but tsnet's logger is private and LogtailWriter returns only io.Writer; it is not a supported per-server configuration seam. Do not rely on type assertions into internal ownership.

Build alternative: ts_omit_logtail selects logtail/logtail_omit.go:4,17-57 no-op implementation (including uploads, NewLogger, Write, Logf and Shutdown); logtail/filch/filch_omit.go similarly no-ops. This changes the entire binary build and requires a new compiled-closure/size/behavior qualification. It is not the measured candidate configuration.

NoLogsNoSupport is a different policy mechanism: envknob/envknob.go:480-484 reads TS_NO_LOGS_NO_SUPPORT; SetNoLogsNoSupport (:498-501) uses Setenv, so violates the no-global-environment-mutation constraint. Also reports an opt-out/technical-support policy to upstream and can interact with tailnet data-plane audit policy (ipn/ipnlocal/local.go:2048). Do not substitute it silently for upload disabling.

Decision needed before enrollment: select pre-Start logtail.Disable (process-wide) or separately qualify build tag; neither currently changed. Confirm intended privacy boundary (logtail only versus all outbound diagnostics) without claiming that upload disabling prevents all telemetry.

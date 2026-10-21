package couchcore

import "github.com/xianxu/pair/cmd/internal/launcher"

// Native query evidence enriches one exact operation. It never replaces launch
// output or participates in authority, lifecycle projection, or operator text.
func (f LifecycleFailure) hasNativeQueryEvidence() bool {
	return len(f.ZellijArgs) > 0 || f.ZellijExecutable != "" || f.ZellijStdout != nil || f.ZellijStderr != nil || f.ZellijStdoutBytes != 0 || f.ZellijStderrBytes != 0 || f.ZellijStdoutTruncated || f.ZellijStderrTruncated || f.ZellijExit != nil || f.ZellijVersion != ""
}

func (f LifecycleFailure) nativeQueryEvidence() launcher.NativeQueryEvidence {
	return launcher.NativeQueryEvidence{Args: f.ZellijArgs, Executable: f.ZellijExecutable, Stdout: f.ZellijStdout, Stderr: f.ZellijStderr, StdoutBytes: f.ZellijStdoutBytes, StderrBytes: f.ZellijStderrBytes, StdoutTruncated: f.ZellijStdoutTruncated, StderrTruncated: f.ZellijStderrTruncated, ExitCode: f.ZellijExit, Version: f.ZellijVersion}
}

func (f *LifecycleFailure) setNativeQueryEvidence(e launcher.NativeQueryEvidence) {
	e = e.Bounded()
	f.ZellijArgs, f.ZellijExecutable = e.Args, e.Executable
	f.ZellijStdout, f.ZellijStderr = e.Stdout, e.Stderr
	f.ZellijStdoutBytes, f.ZellijStderrBytes = e.StdoutBytes, e.StderrBytes
	f.ZellijStdoutTruncated, f.ZellijStderrTruncated = e.StdoutTruncated, e.StderrTruncated
	f.ZellijExit, f.ZellijVersion = e.ExitCode, e.Version
}

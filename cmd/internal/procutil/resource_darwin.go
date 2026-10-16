//go:build darwin && cgo

package procutil

/*
#include <sys/types.h>
#include <sys/sysctl.h>
#include <sys/syslimits.h>
#include <sys/proc.h>
#include <libproc.h>
#include <mach/mach_time.h>
#include <sys/resource.h>
#include <time.h>
#include <errno.h>
#include <stdlib.h>
#include <string.h>

static int resource_table(void *buf, size_t *size) {
 int mib[] = {CTL_KERN, KERN_PROC, KERN_PROC_ALL, 0};
 if (sysctl(mib, 4, buf, size, NULL, 0) == -1) return errno;
 return 0;
}
// A private pointer-free projection, not a declaration of the kernel ABI.
// SDK kinfo_proc field access and union interpretation stay entirely in C.
struct resource_row {
 int pid, ppid, nice, zombie;
 long long sec, usec;
 char name[MAXCOMLEN+1];
};
static void resource_project(const struct kinfo_proc *p, struct resource_row *out) {
 out->pid = p->kp_proc.p_pid;
 out->ppid = p->kp_eproc.e_ppid;
 out->nice = p->kp_proc.p_nice;
 out->zombie = p->kp_proc.p_stat == SZOMB;
 out->sec = p->kp_proc.p_starttime.tv_sec;
 out->usec = p->kp_proc.p_starttime.tv_usec;
 memcpy(out->name, p->kp_proc.p_comm, sizeof(out->name));
 out->name[sizeof(out->name)-1] = 0;
}
static int resource_table_read(struct kinfo_proc *raw, size_t *size, struct resource_row *rows) {
 size_t capacity = *size;
 int err = resource_table(raw, size);
 if (err) return err;
 if (*size > capacity || *size % sizeof(*raw)) return EIO;
 for (size_t i = 0; i < *size / sizeof(*raw); i++) resource_project(&raw[i], &rows[i]);
 return 0;
}
static int resource_identity(int pid, struct resource_row *row, size_t *size) {
 struct kinfo_proc p = {0};
 int mib[] = {CTL_KERN, KERN_PROC, KERN_PROC_PID, pid};
 if (sysctl(mib, 4, &p, size, NULL, 0) == -1) return errno;
 if (*size == sizeof(p)) resource_project(&p, row);
 return 0;
}
static int resource_usage(int pid, struct rusage_info_v2 *p) {
 if (proc_pid_rusage(pid, RUSAGE_INFO_V2, (rusage_info_t *)p) == -1) return errno;
 return 0;
}
static int resource_command(int pid, void *buf, size_t *size) {
 int mib[] = {CTL_KERN, KERN_PROCARGS2, pid};
 if (sysctl(mib, 3, buf, size, NULL, 0) == -1) return errno;
 return 0;
}
static int resource_cpu(struct timespec *p) {
 if (clock_gettime(CLOCK_THREAD_CPUTIME_ID, p) == -1) return errno;
 return 0;
}
*/
import "C"

import (
	"errors"
	"strconv"
	"syscall"
	"unsafe"
)

// The SDK timebase is immutable. Calibrate once at package initialization;
// observation counts cover process/clock calls, not this userspace calibration.
var resourceTimebase = func() C.mach_timebase_info_data_t {
	var tb C.mach_timebase_info_data_t
	if C.mach_timebase_info(&tb) != 0 || tb.denom == 0 {
		return C.mach_timebase_info_data_t{}
	}
	return tb
}()

type darwinResourceCalls struct{}

func platformResourceCalls() resourceCalls { return darwinResourceCalls{} }
func resourceErr(n C.int) error {
	if n != 0 {
		return syscall.Errno(n)
	}
	return nil
}
func (darwinResourceCalls) tableSize() (int, error) {
	var size C.size_t
	err := resourceErr(C.resource_table(nil, &size))
	if err != nil {
		return 0, err
	}
	return int((size + C.sizeof_struct_kinfo_proc - 1) / C.sizeof_struct_kinfo_proc), nil
}
func resourceProcess(p *C.struct_resource_row) Process {
	return Process{PID: int(p.pid), PPID: int(p.ppid), Identity: strconv.FormatInt(int64(p.sec), 10) + "." + strconv.FormatInt(int64(p.usec), 10), Nice: int(p.nice), Name: C.GoString(&p.name[0]), Zombie: p.zombie != 0}
}
func (darwinResourceCalls) tableRead(n int) ([]Process, error) {
	// A non-nil buffer also performs the read for an initially empty table.
	allocated := n
	if allocated == 0 {
		allocated = 1
	}
	raw := make([]C.struct_kinfo_proc, allocated)
	rows := make([]C.struct_resource_row, allocated)
	size := C.size_t(n * C.sizeof_struct_kinfo_proc)
	err := resourceErr(C.resource_table_read(&raw[0], &size, &rows[0]))
	if err != nil {
		return nil, err
	}
	if size%C.sizeof_struct_kinfo_proc != 0 || size > C.size_t(n*C.sizeof_struct_kinfo_proc) {
		return nil, errors.New("invalid process table size")
	}
	out := make([]Process, 0, int(size/C.sizeof_struct_kinfo_proc))
	for i := 0; i < int(size/C.sizeof_struct_kinfo_proc); i++ {
		p := resourceProcess(&rows[i])
		if p.PID > 0 {
			out = append(out, p)
		}
	}
	return out, nil
}
func (darwinResourceCalls) identityRead(pid int) (Process, error) {
	var p C.struct_resource_row
	size := C.size_t(C.sizeof_struct_kinfo_proc)
	err := resourceErr(C.resource_identity(C.int(pid), &p, &size))
	if err != nil {
		return Process{}, err
	}
	if size == 0 {
		return Process{}, nil
	}
	if size != C.sizeof_struct_kinfo_proc {
		return Process{}, errors.New("invalid identity size")
	}
	return resourceProcess(&p), nil
}
func (darwinResourceCalls) usageRead(pid int) (ResourceUsage, error) {
	var u C.struct_rusage_info_v2
	err := resourceErr(C.resource_usage(C.int(pid), &u))
	if err != nil {
		return ResourceUsage{}, err
	}
	if resourceTimebase.denom == 0 {
		return ResourceUsage{}, errors.New("Mach timebase unavailable")
	}
	ticks := uint64(u.ri_user_time) + uint64(u.ri_system_time)
	numer, denom := uint64(resourceTimebase.numer), uint64(resourceTimebase.denom)
	ns := (ticks/denom)*numer + (ticks%denom)*numer/denom
	return ResourceUsage{CPUTimeNS: ns, RSSBytes: uint64(u.ri_resident_size), DiskWriteBytes: uint64(u.ri_diskio_byteswritten)}, nil
}
func (darwinResourceCalls) commandRead(pid int) (CommandInfo, error) {
	info, _, err := darwinProcArgs(pid)
	return info, err
}

// KERN_PROCARGS2 carries the initial environment after argv.
func (darwinResourceCalls) environmentRead(pid int) ([]byte, error) {
	_, env, err := darwinProcArgs(pid)
	return env, err
}
func darwinProcArgs(pid int) (CommandInfo, []byte, error) {
	buf := make([]byte, C.ARG_MAX)
	size := C.size_t(len(buf))
	err := resourceErr(C.resource_command(C.int(pid), unsafe.Pointer(&buf[0]), &size))
	if err != nil {
		return CommandInfo{}, nil, err
	}
	if size > C.size_t(len(buf)) {
		return CommandInfo{}, nil, errors.New("invalid process argument size")
	}
	return ParseDarwinProcArgs(buf[:int(size)])
}
func (darwinResourceCalls) selfCPURead() (uint64, error) {
	var ts C.struct_timespec
	err := resourceErr(C.resource_cpu(&ts))
	return uint64(ts.tv_sec)*1000000000 + uint64(ts.tv_nsec), err
}

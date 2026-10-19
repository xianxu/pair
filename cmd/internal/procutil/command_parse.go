package procutil

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strconv"
	"strings"
)

const maxCommandBytes = 2 << 20

// ParseDarwinProcArgs decodes KERN_PROCARGS2's argc, executable, and argv.
// The returned remainder begins after argv and is reserved for environment
// readers; it is never included in CommandInfo.
func ParseDarwinProcArgs(raw []byte) (CommandInfo, []byte, error) {
	bad := errors.New("malformed Darwin process arguments")
	if len(raw) < 4 || len(raw) > maxCommandBytes {
		return CommandInfo{}, nil, bad
	}
	argc := uint64(binary.NativeEndian.Uint32(raw[:4]))
	raw = raw[4:]
	if argc == 0 || argc > 100000 || argc > uint64(len(raw)) {
		return CommandInfo{}, nil, bad
	}
	end := bytes.IndexByte(raw, 0)
	if end <= 0 {
		return CommandInfo{}, nil, bad
	}
	executable := string(raw[:end])
	raw = raw[end+1:]
	// Darwin pads between the executable and argv[0].
	for len(raw) > 0 && raw[0] == 0 {
		raw = raw[1:]
	}
	if argc > uint64(len(raw)) {
		return CommandInfo{}, nil, bad
	}
	args := make([]string, 0, int(argc))
	for i := uint64(0); i < argc; i++ {
		end = bytes.IndexByte(raw, 0)
		if end < 0 {
			return CommandInfo{}, nil, bad
		}
		args = append(args, string(raw[:end]))
		raw = raw[end+1:]
	}
	return CommandInfo{Executable: executable, Args: args}, raw, nil
}

func parseLinuxCmdline(raw []byte) ([]string, error) {
	if len(raw) == 0 || len(raw) > maxCommandBytes || raw[len(raw)-1] != 0 {
		return nil, errors.New("malformed Linux process command line")
	}
	return strings.Split(string(raw[:len(raw)-1]), "\x00"), nil
}

// Linux stat's comm field may itself contain spaces and parentheses. Field 3
// starts after its final ')'; starttime is field 22, nice field 19.
func parseLinuxStat(pid int, raw []byte) (Process, error) {
	bad := errors.New("malformed Linux process stat")
	if pid <= 0 || len(raw) > maxCommandBytes {
		return Process{}, bad
	}
	s := string(raw)
	start := strings.IndexByte(s, '(')
	end := strings.LastIndexByte(s, ')')
	if start < 1 || end < start {
		return Process{}, bad
	}
	actual, err := strconv.Atoi(strings.TrimSpace(s[:start]))
	if err != nil || actual != pid {
		return Process{}, bad
	}
	fields := strings.Fields(s[end+1:])
	if len(fields) <= 19 || len(fields[0]) != 1 {
		return Process{}, bad
	}
	ppid, err := strconv.Atoi(fields[1])
	if err != nil || ppid < 0 {
		return Process{}, bad
	}
	nice, err := strconv.Atoi(fields[16])
	if err != nil {
		return Process{}, bad
	}
	if _, err := strconv.ParseUint(fields[19], 10, 64); err != nil {
		return Process{}, bad
	}
	ttyNr, err := strconv.ParseInt(fields[4], 10, 64)
	if err != nil {
		return Process{}, bad
	}
	tty := ""
	if ttyNr != 0 {
		tty = strconv.FormatInt(ttyNr, 10)
	}
	return Process{
		PID: pid, PPID: ppid, Identity: fields[19], Nice: nice, Name: s[start+1 : end], TTY: tty,
		// Z is zombie; X/x are Linux's dead states (x on older kernels).
		Zombie: fields[0] == "Z" || fields[0] == "X" || fields[0] == "x",
	}, nil
}

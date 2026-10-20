package couchscratch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// validSlot rejects identities that cannot safely name a private slot directory.
func validSlot(s Slot) bool {
	return s.Repo != "" && s.Repo != "." && s.Repo != ".." && !strings.ContainsAny(s.Repo, "/\\\x00") && s.Identity != "" && s.Number >= 0
}

// decodeOwner validates known fields and duplicate keys while tolerating unknown fields.
func decodeOwner(data []byte) (Slot, error) {
	var s Slot
	d := json.NewDecoder(bytes.NewReader(data))
	tok, err := d.Token()
	if err != nil || tok != json.Delim('{') {
		return s, fmt.Errorf("invalid owner")
	}
	seen := map[string]bool{}
	for d.More() {
		tok, err = d.Token()
		if err != nil {
			return s, err
		}
		key, ok := tok.(string)
		if !ok || seen[key] {
			return s, fmt.Errorf("duplicate owner field")
		}
		seen[key] = true
		var raw json.RawMessage
		if err = d.Decode(&raw); err != nil {
			return s, err
		}
		if (key == "Repo" || key == "Identity" || key == "Number") && bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return s, fmt.Errorf("null owner field %s", key)
		}
		switch key {
		case "Repo":
			err = json.Unmarshal(raw, &s.Repo)
		case "Identity":
			err = json.Unmarshal(raw, &s.Identity)
		case "Number":
			err = json.Unmarshal(raw, &s.Number)
		}

		if err != nil {
			return s, err
		}
	}
	if _, err = d.Token(); err != nil {
		return s, err
	}
	var extra any
	if d.Decode(&extra) != io.EOF || !validSlot(s) || !seen["Repo"] || !seen["Identity"] || !seen["Number"] {
		return s, fmt.Errorf("invalid owner declaration")
	}
	return s, nil
}

// PrepareSlot returns only the temporary-directory overlays. Failure preserves
// inherited values, including when maintenance currently owns the manager.
// An unmounted volume is never created on the host disk.
func (m *Manager) PrepareSlot(s Slot) []string {
	return m.PrepareResolvedSlot(func() (Slot, error) { return s, nil })
}

// PrepareResolvedSlot first checks availability without waiting for maintenance,
// then resolves the slot identity only if scratch can be adopted. A resolver
// error preserves inherited temporary values and emits one bounded diagnostic.
func (m *Manager) PrepareResolvedSlot(resolve func() (Slot, error)) []string {
	if !m.mu.TryLock() {
		m.note("scratch unavailable: maintenance or another preparation is active")
		return nil
	}
	defer m.mu.Unlock()
	p := m.options.Probe()
	if !ChooseScratch(p) {
		if p.Mounted {
			m.note("scratch unavailable: space floor or probe failure")
		}
		return nil
	}
	if resolve == nil {
		m.note("scratch unavailable: missing slot identity resolver")
		return nil
	}
	s, err := resolve()
	if err != nil {
		m.note("scratch unavailable: " + err.Error())
		return nil
	}
	if !validSlot(s) {
		m.note("invalid scratch slot identity")
		return nil
	}
	env, err := prepare(p.Root, s)
	if err != nil {
		m.note("scratch unavailable: " + err.Error())
		return nil
	}
	return env
}

// prepare creates scratch directories under an exclusively locked immutable owner declaration.
func prepare(root string, s Slot) ([]string, error) {
	fd, err := openDir(unix.AT_FDCWD, root)
	if err != nil {
		return nil, err
	}
	defer unix.Close(fd)
	book, err := mkdirDir(fd, "couch-scratch")
	if err != nil {
		return nil, err
	}
	defer unix.Close(book)
	owners, err := mkdirDir(book, "owners")
	if err != nil {
		return nil, err
	}
	defer unix.Close(owners)
	lock, err := unix.Openat(owners, ".lock", unix.O_CREAT|unix.O_RDWR|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0600)
	if err != nil {
		return nil, err
	}
	defer unix.Close(lock)
	var lockStat unix.Stat_t
	if err = unix.Fstat(lock, &lockStat); err != nil {
		return nil, err
	}
	if lockStat.Mode&unix.S_IFMT != unix.S_IFREG || lockStat.Mode&07777 != 0600 || int(lockStat.Uid) != os.Getuid() {
		return nil, fmt.Errorf("unsafe owner lock")
	}
	if err = unix.Flock(lock, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return nil, err
	}
	defer unix.Flock(lock, unix.LOCK_UN)
	name := slotName(s)
	decl := name + ".json"
	have, err := readOwnerAt(owners, decl, &Result{})
	if err == nil {
		if have != s {
			return nil, fmt.Errorf("scratch owner conflict")
		}
	} else if err == unix.ENOENT {
		data, _ := json.Marshal(s)
		f, err := unix.Openat(owners, decl, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
		if err != nil {
			return nil, err
		}
		file := os.NewFile(uintptr(f), decl)
		_, e := file.Write(data)
		ce := file.Close()
		if e != nil {
			return nil, e
		}
		if ce != nil {
			return nil, ce
		}
	} else {
		return nil, err
	}
	tmp, err := openSharedTmp(fd, true, nil)
	if err != nil {
		return nil, err
	}
	defer unix.Close(tmp)
	slot, err := mkdirDir(tmp, name)
	if err != nil {
		return nil, err
	}
	defer unix.Close(slot)
	probe, err := unix.Openat(slot, ".write-probe", unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if err != nil {
		return nil, err
	}
	unix.Close(probe)
	if err = unix.Unlinkat(slot, ".write-probe", 0); err != nil {
		return nil, err
	}
	path := filepath.Join(root, "tmp", name)
	return []string{"TMPDIR=" + path, "GOTMPDIR=" + path, "CLAUDE_CODE_TMPDIR=" + path}, nil
}

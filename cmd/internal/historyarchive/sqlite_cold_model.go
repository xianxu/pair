package historyarchive

// coldFileStamp records the identity and mutation metadata observed around a copy.
// pair:m5-concept pure
type coldFileStamp struct {
	Device, Inode                            uint64
	Size                                     int64
	Mode                                     uint32
	MtimeSec, MtimeNsec, CtimeSec, CtimeNsec int64
}

// coldSourceStamp includes the containing directory to detect sidecar churn.
// pair:m5-concept pure
type coldSourceStamp struct {
	DB, Parent coldFileStamp
	Sidecars   uint8
}

// coldWALHeader recognizes only a complete SQLite header in WAL mode.
func coldWALHeader(header []byte) bool {
	return len(header) >= 100 && string(header[:16]) == "SQLite format 3\x00" &&
		header[18] == 2 && header[19] == 2
}

// sameColdSource proves matching observations, not exclusion of concurrent writers.
func sameColdSource(before, after coldSourceStamp) bool {
	return before.Sidecars == 0 && after.Sidecars == 0 && before == after
}

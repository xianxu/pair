package historyarchive

import "testing"

func TestColdWALHeader(t *testing.T) {
	header := make([]byte, 100)
	copy(header, "SQLite format 3\x00")
	header[18], header[19] = 2, 2
	for n := 0; n < 100; n++ {
		if coldWALHeader(header[:n]) {
			t.Fatalf("accepted truncated header of %d bytes", n)
		}
	}
	if !coldWALHeader(header) || !coldWALHeader(append(header, 99)) {
		t.Fatal("rejected complete WAL header")
	}
	for i := 0; i < 16; i++ {
		for b := 0; b < 256; b++ {
			if byte(b) == header[i] {
				continue
			}
			mutated := append([]byte(nil), header...)
			mutated[i] = byte(b)
			if coldWALHeader(mutated) {
				t.Fatalf("accepted signature mutation at %d to %d", i, b)
			}
		}
	}
	for _, i := range []int{18, 19} {
		for b := 0; b < 256; b++ {
			mutated := append([]byte(nil), header...)
			mutated[i] = byte(b)
			if got := coldWALHeader(mutated); got != (b == 2) {
				t.Fatalf("version byte %d=%d: got %v", i, b, got)
			}
		}
	}
	// This is a format/mode discriminator, not a schema or page validator.
	for i := 16; i < 100; i++ {
		if i == 18 || i == 19 {
			continue
		}
		mutated := append([]byte(nil), header...)
		mutated[i] = 255
		if !coldWALHeader(mutated) {
			t.Fatalf("rejected unrelated header byte %d", i)
		}
	}
}

func TestSameColdSourceRequiresEveryStampAndNoSidecars(t *testing.T) {
	base := coldSourceStamp{
		DB: coldFileStamp{Device: 1, Inode: 2, Size: 4096, Mode: 0600,
			MtimeSec: 3, MtimeNsec: 4, CtimeSec: 5, CtimeNsec: 6},
		Parent: coldFileStamp{Device: 7, Inode: 8, Size: 128, Mode: 0700,
			MtimeSec: 9, MtimeNsec: 10, CtimeSec: 11, CtimeNsec: 12},
	}
	if !sameColdSource(base, base) {
		t.Fatal("unchanged source rejected")
	}
	changes := []func(*coldFileStamp){
		func(s *coldFileStamp) { s.Device++ },
		func(s *coldFileStamp) { s.Inode++ },
		func(s *coldFileStamp) { s.Size++ },
		func(s *coldFileStamp) { s.Mode++ },
		func(s *coldFileStamp) { s.MtimeSec++ },
		func(s *coldFileStamp) { s.MtimeNsec++ },
		func(s *coldFileStamp) { s.CtimeSec++ },
		func(s *coldFileStamp) { s.CtimeNsec++ },
	}
	for _, parent := range []bool{false, true} {
		for i, change := range changes {
			altered := base
			if parent {
				change(&altered.Parent)
			} else {
				change(&altered.DB)
			}
			if sameColdSource(base, altered) || sameColdSource(altered, base) {
				t.Fatalf("accepted change %d (parent=%v)", i, parent)
			}
		}
	}
	for bits := 1; bits < 256; bits++ {
		altered := base
		altered.Sidecars = uint8(bits)
		if sameColdSource(base, altered) || sameColdSource(altered, base) || sameColdSource(altered, altered) {
			t.Fatalf("accepted sidecars %d", bits)
		}
	}
}

func FuzzColdWALHeader(f *testing.F) {
	f.Add([]byte("SQLite format 3\x00"))
	header := make([]byte, 100)
	copy(header, "SQLite format 3\x00")
	header[18], header[19] = 2, 2
	f.Add(header)
	f.Fuzz(func(t *testing.T, input []byte) {
		want := len(input) >= 100
		if want {
			want = string(input[:16]) == "SQLite format 3\x00" && input[18] == 2 && input[19] == 2
		}
		if coldWALHeader(input) != want {
			t.Fatalf("unexpected header admission for %x", input)
		}
	})
}

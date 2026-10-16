package procutil

import "bytes"

// ParseEnvironment picks keys out of NUL-separated KEY=VALUE entries; the
// rest of the environment is never retained (pair#606).
// pair:m5-concept pure
func ParseEnvironment(raw []byte, keys ...string) map[string]string {
	want := make(map[string]bool, len(keys))
	for _, k := range keys {
		want[k] = true
	}
	out := map[string]string{}
	for _, entry := range bytes.Split(raw, []byte{0}) {
		eq := bytes.IndexByte(entry, '=')
		if eq <= 0 {
			continue
		}
		if k := string(entry[:eq]); want[k] {
			if _, seen := out[k]; !seen {
				out[k] = string(entry[eq+1:])
			}
		}
	}
	return out
}

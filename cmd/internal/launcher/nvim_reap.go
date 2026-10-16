package launcher

import "path/filepath"

// NvimReapRow is one embedded-nvim candidate, assembled from a single process
// snapshot plus one evidence source (a pid sidecar, argv, or a changelog
// record). Every field is untrusted input: the reaper reads it, decides, and
// re-fences the identity at signal time.
//
// pair:m5-concept pure
type NvimReapRow struct {
	Kind  string
	Owner string // a tag, or a changelog artifact base
	PID   int
	PPID  int
	// Identity is the kernel start stamp; empty means the read failed.
	Identity string
	Name     string
	Args     []string
	// ArgsKnown is false when argv could not be read, which is not the same as
	// reading an empty argv.
	ArgsKnown bool
	// OwnerLive is false on the explicit quit path: quitting authorizes the reap.
	OwnerLive bool
	// ExpectedArgPath is the scope path argv must carry exactly; "" for kinds
	// whose argv carries no scope path (review).
	ExpectedArgPath string
}

// ClassifyNvimReap decides one candidate. It kills only when the pid is
// positive, the owner is identified, the identity is known, the short name is
// nvim, argv is known and carries an exact "--embed" field, argv carries
// ExpectedArgPath exactly when one is expected, and the owner is dead or the
// process is reparented to launchd. Every skip returns a reason; nothing is
// signalled on missing evidence.
func ClassifyNvimReap(row NvimReapRow) (kill bool, reason string) {
	if row.PID <= 0 {
		return false, "no pid"
	}
	if row.Owner == "" {
		return false, "no owner"
	}
	if row.Identity == "" {
		return false, "identity unknown"
	}
	if filepath.Base(row.Name) != "nvim" {
		return false, "not nvim"
	}
	if !row.ArgsKnown {
		return false, "argv unknown"
	}
	if !argvCarriesField(row.Args, "--embed") {
		return false, "no --embed field"
	}
	if row.ExpectedArgPath != "" && !argvCarriesField(row.Args, row.ExpectedArgPath) {
		return false, "argv serves another path"
	}
	if row.OwnerLive {
		if row.PPID != 1 {
			return false, "owner live"
		}
		return true, "reparented to launchd"
	}
	return true, "owner dead"
}

// argvCarriesField compares whole fields, never substrings: `--embed-x` and a
// path containing `--embed` are not an embed flag (#587's classifier rule).
func argvCarriesField(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

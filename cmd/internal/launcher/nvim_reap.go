package launcher

import "path/filepath"

// The reap decision's vocabulary. ClassifyNvimReap returns one of these for
// every candidate, and the sidecar rule below reads them back, so a reason is
// never re-spelled at a call site.
const (
	reapReasonNoPID           = "no pid"
	reapReasonNoOwner         = "no owner"
	reapReasonIdentityUnknown = "identity unknown"
	reapReasonNotNvim         = "not nvim"
	reapReasonArgvUnknown     = "argv unknown"
	reapReasonNoEmbed         = "no --embed field"
	reapReasonOtherPath       = "argv serves another path"
	reapReasonOtherTag        = "argv serves another tag"
	reapReasonOtherAsset      = "argv runs another config"
	reapReasonOwnerLive       = "owner live"
	reapReasonReparented      = "reparented to launchd"
	reapReasonOwnerDead       = "owner dead"
)

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
	// Attribution: what argv must carry for this pid to be the process the
	// evidence names. A kind states as much as its argv allows, and a candidate
	// that cannot be attributed is never killed.
	//
	//   ExpectedArgPath   an exact argv field: the artifact the embed serves,
	//                     for a kind whose path is deterministic (draft).
	//   ExpectArgvTag     argv must be attributed to Owner by the scope's parser,
	//                     for a kind whose path embeds a varying component
	//                     (scrollback carries the agent). ArgvTag is that
	//                     attribution; "" means argv carried no scope path.
	//   ExpectedAssetBase the base name of the pair nvim asset argv must carry,
	//                     for a kind whose argv names no scope path at all
	//                     (review serves a file anywhere in the user's tree).
	//                     Base, not path: pair's install root differs per
	//                     checkout, and an embed outlives the checkout that
	//                     spawned it.
	ExpectedArgPath   string
	ExpectArgvTag     bool
	ArgvTag           string
	ExpectedAssetBase string
}

// ClassifyNvimReap decides one candidate. It kills only when the pid is
// positive, the owner is identified, the identity is known, the short name is
// nvim, argv is known and carries an exact "--embed" field, argv satisfies every
// attribution the kind states, and the owner is dead or the process is
// reparented to launchd. Every skip returns a reason; nothing is signalled on
// missing evidence.
func ClassifyNvimReap(row NvimReapRow) (kill bool, reason string) {
	if row.PID <= 0 {
		return false, reapReasonNoPID
	}
	if row.Owner == "" {
		return false, reapReasonNoOwner
	}
	if row.Identity == "" {
		return false, reapReasonIdentityUnknown
	}
	if filepath.Base(row.Name) != "nvim" {
		return false, reapReasonNotNvim
	}
	if !row.ArgsKnown {
		return false, reapReasonArgvUnknown
	}
	if !argvCarriesField(row.Args, "--embed") {
		return false, reapReasonNoEmbed
	}
	if row.ExpectedArgPath != "" && !argvCarriesField(row.Args, row.ExpectedArgPath) {
		return false, reapReasonOtherPath
	}
	if row.ExpectArgvTag && row.ArgvTag != row.Owner {
		return false, reapReasonOtherTag
	}
	if row.ExpectedAssetBase != "" && !argvCarriesAssetBase(row.Args, row.ExpectedAssetBase) {
		return false, reapReasonOtherAsset
	}
	if row.OwnerLive {
		if row.PPID != 1 {
			return false, reapReasonOwnerLive
		}
		return true, reapReasonReparented
	}
	return true, reapReasonOwnerDead
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

func argvCarriesAssetBase(args []string, base string) bool {
	for _, arg := range args {
		if filepath.Base(arg) == base {
			return true
		}
	}
	return false
}

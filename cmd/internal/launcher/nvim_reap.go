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
	reapReasonNoAttribution   = "attribution unknown"
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
	// Attribution has to name the *owner*, not merely the kind (#583): the pid a
	// dead tag's record names can be held by another tag's live embed by now, so
	// evidence every tag's embed shares authorizes no kill. ExpectArgvPath and
	// ExpectArgvTag are the two owner-naming forms; a row stating neither is
	// unattributable.
	//
	//   ExpectArgvPath    argv must carry ExpectedArgPath as an exact field: the
	//                     artifact the embed serves. Deterministic for draft (the
	//                     scope's own draft path) and for a changelog viewer (its
	//                     .md); for review, whose file is anywhere in the user's
	//                     tree, it is the file this tag's own review state names
	//                     -- so an empty value means the state named none, which
	//                     is missing evidence rather than a pass.
	//   ExpectArgvTag     argv must be attributed to Owner by the scope's parser,
	//                     for a kind whose path embeds a varying component
	//                     (scrollback carries the agent). ArgvTag is that
	//                     attribution; "" means argv carried no scope path.
	//   ExpectedAssetBase the base name of the pair nvim asset argv must carry.
	//                     Corroboration only: every tag's review pane runs the
	//                     same config, so it names the kind and never the owner.
	//                     Base, not path: pair's install root differs per
	//                     checkout, and an embed outlives the checkout that
	//                     spawned it.
	ExpectArgvPath    bool
	ExpectedArgPath   string
	ExpectArgvTag     bool
	ArgvTag           string
	ExpectedAssetBase string
}

// ClassifyNvimReap decides one candidate. It kills only when the pid is
// positive, the owner is identified, the identity is known, the short name is
// nvim, argv is known and carries an exact "--embed" field, argv carries
// attribution naming that owner, and the owner is dead or the process is
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
	attributed := false
	if row.ExpectArgvPath {
		if row.ExpectedArgPath == "" {
			return false, reapReasonNoAttribution
		}
		if !argvCarriesField(row.Args, row.ExpectedArgPath) {
			return false, reapReasonOtherPath
		}
		attributed = true
	}
	if row.ExpectArgvTag {
		if row.ArgvTag != row.Owner {
			return false, reapReasonOtherTag
		}
		attributed = true
	}
	if !attributed {
		return false, reapReasonNoAttribution
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

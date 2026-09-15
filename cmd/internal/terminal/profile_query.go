package terminal

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/x/ansi"
	vt "github.com/charmbracelet/x/vt"
)

const PairTERM = "pair-vt-256color"

func DefaultProfile() Profile { return Profile{TERM: PairTERM} }

func (p Profile) Validate() error {
	if p.TERM != PairTERM {
		return fmt.Errorf("terminal: unsupported TERM %q", p.TERM)
	}
	if p.ClipboardRead || p.Graphics {
		return errors.New("terminal: graphics and clipboard reads are not implemented")
	}
	return nil
}

// profileCapabilities is the common authority for the compiled terminfo source
// and XTGETTCAP values. No host terminfo entry or ambient terminal is consulted.
// b/n/s distinguish boolean, numeric and string terminfo capabilities.
var profileCapabilities = []struct {
	name  string
	kind  byte
	value string
}{
	{"BD", 's', "\u001b[?2004l"},
	{"BE", 's', "\u001b[?2004h"},
	{"PE", 's', "\u001b[201~"},
	{"PS", 's', "\u001b[200~"},
	{"RGB", 'b', "1"},
	{"Se", 's', "\u001b[0 q"},
	{"Ss", 's', "\u001b[%p1%d q"},
	{"Sync", 's', "\u001b[?2026%?%p1%th%el%;"},
	{"XM", 's', "%?%p1%t\u001b[?1000h\u001b[?1006h%e\u001b[?1000l\u001b[?1006l%;"},
	{"acsc", 's', "``aaffggiijjkkllmmnnooppqqrrssttuuvvwwxxyyzz{{||}}~~"},
	{"am", 'b', "1"},
	{"bel", 's', "\u0007"},
	{"blink", 's', "\u001b[5m"},
	{"bold", 's', "\u001b[1m"},
	{"cbt", 's', "\u001b[Z"},
	{"civis", 's', "\u001b[?25l"},
	{"clear", 's', "\u001b[H\u001b[2J"},
	{"cnorm", 's', "\u001b[?25h"},
	{"colors", 'n', "256"},
	{"cols", 'n', "80"},
	{"cr", 's', "\r"},
	{"csr", 's', "\u001b[%i%p1%d;%p2%dr"},
	{"cub", 's', "\u001b[%p1%dD"},
	{"cub1", 's', "\b"},
	{"cud", 's', "\u001b[%p1%dB"},
	{"cud1", 's', "\n"},
	{"cuf", 's', "\u001b[%p1%dC"},
	{"cuf1", 's', "\u001b[C"},
	{"cup", 's', "\u001b[%i%p1%d;%p2%dH"},
	{"cuu", 's', "\u001b[%p1%dA"},
	{"cuu1", 's', "\u001b[A"},
	{"dch", 's', "\u001b[%p1%dP"},
	{"dim", 's', "\u001b[2m"},
	{"dl", 's', "\u001b[%p1%dM"},
	{"ech", 's', "\u001b[%p1%dX"},
	{"ed", 's', "\u001b[J"},
	{"el", 's', "\u001b[K"},
	{"el1", 's', "\u001b[1K"},
	{"home", 's', "\u001b[H"},
	{"hpa", 's', "\u001b[%i%p1%dG"},
	{"ht", 's', "\t"},
	{"hts", 's', "\u001bH"},
	{"ich", 's', "\u001b[%p1%d@"},
	{"il", 's', "\u001b[%p1%dL"},
	{"ind", 's', "\n"},
	{"indn", 's', "\u001b[%p1%dS"},
	{"invis", 's', "\u001b[8m"},
	{"it", 'n', "8"},
	{"kbs", 's', "\u007f"},
	{"kcbt", 's', "\u001b[Z"},
	{"kcub1", 's', "\u001bOD"},
	{"kcud1", 's', "\u001bOB"},
	{"kcuf1", 's', "\u001bOC"},
	{"kcuu1", 's', "\u001bOA"},
	{"kdch1", 's', "\u001b[3~"},
	{"kend", 's', "\u001b[F"},
	{"kent", 's', "\u001bOM"},
	{"kf1", 's', "\u001bOP"},
	{"kf10", 's', "\u001b[21~"},
	{"kf11", 's', "\u001b[23~"},
	{"kf12", 's', "\u001b[24~"},
	{"kf2", 's', "\u001bOQ"},
	{"kf3", 's', "\u001bOR"},
	{"kf4", 's', "\u001bOS"},
	{"kf5", 's', "\u001b[15~"},
	{"kf6", 's', "\u001b[17~"},
	{"kf7", 's', "\u001b[18~"},
	{"kf8", 's', "\u001b[19~"},
	{"kf9", 's', "\u001b[20~"},
	{"khome", 's', "\u001b[H"},
	{"kich1", 's', "\u001b[2~"},
	{"km", 'b', "1"},
	{"kmous", 's', "\u001b[<"},
	{"knp", 's', "\u001b[6~"},
	{"kpp", 's', "\u001b[5~"},
	{"lines", 'n', "24"},
	{"msgr", 'b', "1"},
	{"op", 's', "\u001b[39;49m"},
	{"pairs", 'n', "32767"},
	{"rc", 's', "\u001b8"},
	{"rev", 's', "\u001b[7m"},
	{"ri", 's', "\u001bM"},
	{"rin", 's', "\u001b[%p1%dT"},
	{"ritm", 's', "\u001b[23m"},
	{"rmacs", 's', "\u001b(B"},
	{"rmcup", 's', "\u001b[?1049l"},
	{"rmkx", 's', "\u001b[?1l\u001b>"},
	{"rmso", 's', "\u001b[27m"},
	{"rmul", 's', "\u001b[24m"},
	{"sc", 's', "\u001b7"},
	{"setab", 's', "\u001b[48;5;%p1%dm"},
	{"setaf", 's', "\u001b[38;5;%p1%dm"},
	{"setrgbb", 's', "\u001b[48;2;%p1%d;%p2%d;%p3%dm"},
	{"setrgbf", 's', "\u001b[38;2;%p1%d;%p2%d;%p3%dm"},
	{"sgr0", 's', "\u001b(B\u001b[0m"},
	{"sitm", 's', "\u001b[3m"},
	{"smacs", 's', "\u001b(0"},
	{"smcup", 's', "\u001b[?1049h"},
	{"smkx", 's', "\u001b[?1h\u001b="},
	{"smso", 's', "\u001b[7m"},
	{"smul", 's', "\u001b[4m"},
	{"tbc", 's', "\u001b[3g"},
	{"vpa", 's', "\u001b[%i%p1%dd"},
	{"xenl", 'b', "1"},
}

// TerminfoSource is compiled with tic during packaging, never by a terminal
// runtime. The committed source must match this deterministic contract.
func (p Profile) TerminfoSource() string {
	if p.Validate() != nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("# Generated from terminal.Profile.TerminfoSource; keep the contract test passing.\n# Deliberately excludes palette mutation, printer, graphics and clipboard reads.\npair-vt-256color|Pair virtual terminal text profile,\n")
	escape := strings.NewReplacer("\\", "\\\\", "\x1b", "\\E", "\a", "^G", "\r", "^M", "\n", "^J", "\b", "^H", "\t", "^I", "\x7f", "^?", ",", "\\,", ":", "\\:", "^", "\\^")
	for _, cap := range profileCapabilities {
		b.WriteByte('\t')
		b.WriteString(cap.name)
		switch cap.kind {
		case 'n':
			b.WriteByte('#')
			b.WriteString(cap.value)
		case 's':
			b.WriteByte('=')
			b.WriteString(escape.Replace(cap.value))
		}
		b.WriteString(",\n")
	}
	return b.String()
}

func (p Profile) capability(name string) (string, bool) {
	switch name {
	case "TN", "name":
		return p.TERM, true
	case "Co":
		name = "colors"
	}
	// Traditional termcap names for the key capabilities in the same profile.
	aliases := map[string]string{"ku": "kcuu1", "kd": "kcud1", "kr": "kcuf1", "kl": "kcub1", "kh": "khome", "@7": "kend", "kI": "kich1", "kD": "kdch1", "kP": "kpp", "kN": "knp", "kb": "kbs", "kB": "kcbt", "@8": "kent", "k1": "kf1", "k2": "kf2", "k3": "kf3", "k4": "kf4", "k5": "kf5", "k6": "kf6", "k7": "kf7", "k8": "kf8", "k9": "kf9", "k;": "kf10", "F1": "kf11", "F2": "kf12"}
	if alias, ok := aliases[name]; ok {
		name = alias
	}
	for _, cap := range profileCapabilities {
		if cap.name == name {
			return cap.value, true
		}
	}
	return "", false
}

// Profile queries are local protocol replies, never forwarded to the parent.
// The supplied writer belongs to the endpoint's ordered input stream.
func (p Profile) InstallQueries(target interface {
	RegisterCsiHandler(int, vt.CsiHandler)
	RegisterDcsHandler(int, vt.DcsHandler)
}, replies io.Writer) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if target == nil || replies == nil {
		return errors.New("terminal: profile queries need target and reply writer")
	}
	da := func(reply string) vt.CsiHandler {
		return func(params ansi.Params) bool {
			n, _, _ := params.Param(0, 0)
			if n == 0 && len(params) <= 1 {
				_, _ = io.WriteString(replies, reply)
			}
			return true // Do not fall back to the backend's broader VT220 advertisement.
		}
	}
	// VT100 advanced-video text baseline; color and negotiated extras are
	// described by terminfo/feature queries, not a fictitious VT220 feature list.
	target.RegisterCsiHandler('c', da("\x1b[?1;2c"))
	target.RegisterCsiHandler(ansi.Command('>', 0, 'c'), da("\x1b[>0;1;0c"))
	target.RegisterDcsHandler(ansi.Command(0, '+', 'q'), func(params ansi.Params, data []byte) bool {
		unsupported := "\x1bP0+r\x1b\\"
		if len(params) != 0 || len(data) == 0 || len(data) > 1024 {
			_, _ = io.WriteString(replies, unsupported)
			return true
		}
		names := strings.Split(string(data), ";")
		if len(names) > 32 {
			_, _ = io.WriteString(replies, unsupported)
			return true
		}
		var pairs []string
		for _, encoded := range names {
			decoded, err := hex.DecodeString(encoded)
			value, ok := p.capability(string(decoded))
			if err != nil || !ok {
				if len(pairs) > 0 {
					_, _ = io.WriteString(replies, "\x1bP1+r"+strings.Join(pairs, ";")+"\x1b\\")
				}
				_, _ = io.WriteString(replies, unsupported)
				return true
			}
			pairs = append(pairs, encoded+"="+hex.EncodeToString([]byte(value)))
		}
		_, _ = io.WriteString(replies, "\x1bP1+r"+strings.Join(pairs, ";")+"\x1b\\")
		return true
	})
	return nil
}

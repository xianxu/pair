module github.com/xianxu/pair

go 1.27.3

require (
	github.com/charmbracelet/ultraviolet v0.0.0-20260811164956-006e29f97886
	github.com/charmbracelet/x/ansi v0.11.8
	github.com/charmbracelet/x/vt v0.0.0-20260510215043-e3181689be6b
	github.com/coreos/go-oidc/v3 v3.21.0
	github.com/creack/pty v1.1.24
	github.com/go-jose/go-jose/v4 v4.1.4
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	github.com/xianxu/ariadne v0.0.0-20261018104241-81c832d5e000
	golang.org/x/image v0.46.0
	golang.org/x/net v0.59.0
	golang.org/x/sys v0.48.0
	golang.org/x/term v0.46.0
)

require (
	github.com/charmbracelet/colorprofile v0.4.3 // indirect
	github.com/charmbracelet/x/exp/ordered v0.1.0 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/charmbracelet/x/termios v0.1.1 // indirect
	github.com/charmbracelet/x/windows v0.2.2 // indirect
	github.com/clipperhouse/displaywidth v0.11.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.4.1 // indirect
	github.com/mattn/go-runewidth v0.0.24 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/xo/terminfo v1.0.0 // indirect
	golang.org/x/oauth2 v0.37.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
)

replace github.com/charmbracelet/x/vt => ./third_party/vt

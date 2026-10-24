package repolint

import (
	"github.com/xianxu/pair/internal/reporoot"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/ariadne/pkg/golint"
)

func TestZellijNativeAdmissionCatalog(t *testing.T) {
	for _, tc := range []struct {
		name, path, source string
		want               int
	}{
		{"source hook outside owner", "cmd/internal/new/native.go", `package new;import "github.com/xianxu/pair/cmd/internal/zellijquery";func bad(s zellijquery.Source){s.Run(cmd)}`, 1},
		{"raw collector", "cmd/internal/new/native.go", `package new;import "github.com/xianxu/pair/cmd/internal/zellijexec";func bad(){zellijexec.Run(cmd)}`, 1},
		{"dynamic native variable", "cmd/internal/new/native.go", `package new;import "os/exec";func bad(){path:="zellij";exec.CommandContext(ctx,path,args...)}`, 1},
		{"direct process replacement", "cmd/internal/new/native.go", `package new;import "syscall";func bad(){syscall.Exec("/opt/bin/zellij",argv,env)}`, 1},
		{"mutation", "cmd/internal/new/native.go", `package new; import "os/exec"; func bad(){exec.Command("zellij","action","rename-pane","x")}`, 1},
		{"absolute mutation", "cmd/internal/new/native.go", `package new; import x "os/exec"; func bad(){x.CommandContext(ctx,"/opt/bin/zellij","kill-session",name)}`, 1},
		{"version", "cmd/internal/new/native.go", `package new; import "os/exec"; func ok(){exec.Command("zellij","--version")}`, 0},
		{"guarded mutation", "cmd/internal/new/native.go", `package new; import "os/exec"; import "github.com/xianxu/pair/cmd/internal/zellijquery"; func ok(){zellijquery.WithProbe(ctx,env,func(ctx context.Context)error{exec.Command("zellij","action","write","1");return nil})}`, 0},
		{"unrelated guard", "cmd/internal/new/native.go", `package new; import "os/exec"; func bad(){WithProbe(ctx,env,func(){exec.Command("zellij","action","write","1")})}`, 1},
		{"generated shell", "cmd/internal/new/native.go", "package new; const shell = `echo begin; zellij action rename-pane x; exec pair term`", 1},
		{"helper shell", "cmd/internal/new/native.go", "package new; const shell = `pair internal zellij-command -- action rename-pane x; exec pair term`", 0},
		{"new call in creator file", "cmd/internal/launcher/session_handoff_os.go", `package launcher; import "os/exec";func other(){exec.Command("zellij","action","list-panes")}`, 1},
		{"source injection bypass", "cmd/internal/zellijquery/query.go", `package zellijquery;func (s Source) readAttempt(){run:=s.Run;run(cmd)}`, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			findings, err := golint.Check(tc.path, []byte(tc.source), pairCatalogFor(tc.path))
			if err != nil {
				t.Fatal(err)
			}
			n := 0
			for _, f := range findings {
				if f.Rule == "zellij-native-admission" {
					n++
				}
			}
			if n != tc.want {
				t.Fatalf("findings=%v; native admission count=%d want %d", findings, n, tc.want)
			}
		})
	}
}

func TestZellijNativeSourceControls(t *testing.T) {
	for _, tc := range []struct {
		name, path, source string
		want               bool
	}{
		{"layout", "zellij/layouts/new.kdl", `args "-c" "zellij action list-panes"`, true},
		{"indented shell", "bin/new.sh", "  zellij action write 1", true},
		{"quoted shell", "bin/new.sh", `"zellij" action write 1`, true},
		{"dynamic shell argv", "bin/new.sh", `zellij "$@"`, true},
		{"new action in reproducer", "scripts/repro/zellij-false-miss.sh", `zellij action write 1`, true},
		{"unclassified interactive shell", "bin/new.sh", `zellij attach test`, true},
		{"version shell", "bin/new.sh", `zellij --version`, false},
		{"shell", "bin/new.sh", `zellij --session "$s" action rename-pane x`, true},
		{"shell pipeline", "bin/new.sh", `echo x; /opt/bin/zellij action list-panes`, true},
		{"helper", "zellij/layouts/new.kdl", `args "-c" "pair internal zellij-command -- action list-panes"`, false},
		{"lua direct table", "nvim/new.lua", `vim.system({'zellij','action','write','1'})`, true},
		{"lua string", "nvim/new.lua", `vim.fn.system('zellij action list-panes')`, true},
		{"lua variable", "nvim/new.lua", "local argv = { 'zellij', 'action', 'list-panes' }\nvim.fn.system(argv)", true},
		{"lua guard", "nvim/new.lua", "local argv = { 'zellij', 'action', 'list-panes' }\nPairZellijTrace.system(argv)", false},
		{"adapter bypass", "nvim/zellij_trace.lua", `vim.system(argv,{text=true})`, true},
		{"adapter guard", "nvim/zellij_trace.lua", `vim.system(M.command(argv),{text=true})`, false},
		{"comment", "nvim/new.lua", `-- vim.fn.system('zellij action list-panes')`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			findings := zellijNativeSourceFindings(tc.path, []byte(tc.source))
			if (len(findings) > 0) != tc.want {
				t.Fatalf("findings=%v want finding=%v", findings, tc.want)
			}
			for _, f := range findings {
				if !strings.Contains(f.Finding.Message, "#866") {
					t.Fatal("missing incident")
				}
			}
		})
	}
}

func TestZellijAdmissionProductionMutationControls(t *testing.T) {
	root, err := reporoot.Current()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, path, old, replacement string }{
		{"source acquire removed", "cmd/internal/zellijquery/query.go", "lease, err := acquire(ctx, cmd.Env)", "lease, err := unguarded(ctx, cmd.Env)"},
		{"source hook before acquire", "cmd/internal/zellijquery/query.go", "lease, err := acquire(ctx, cmd.Env)", "result.ProcessErr = s.Run(cmd)\nlease, err := acquire(ctx, cmd.Env)"},
		{"source release removed", "cmd/internal/zellijquery/query.go", "lease.Close()", "discard(lease)"},
		{"native adapter admission removed", "cmd/internal/zellijquery/command.go", "return withNativeAdmission(ctx, snapshot, nativeCommandReads(args),", "return unguarded(ctx, snapshot, nativeCommandReads(args),"},
		{"lua adapter bypass", "nvim/zellij_trace.lua", "vim.system(M.command(argv),", "vim.system(argv,"},
		{"restore adapter bypass", "nvim/review/restore_client.lua", "cmd=zellij_trace.command(cmd,self.opts.pair_bin())", "cmd=cmd"},
		{"layout bypass", "zellij/layouts/main-2.kdl", "pair internal zellij-command -- action rename-pane", "zellij action rename-pane"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original, err := os.ReadFile(filepath.Join(root, tc.path))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Count(string(original), tc.old) != 1 {
				t.Fatalf("production seam moved: %q", tc.old)
			}
			scan := func(source []byte) int {
				if !strings.HasSuffix(tc.path, ".go") {
					return len(zellijNativeSourceFindings(tc.path, source))
				}
				findings, err := golint.Check(tc.path, source, pairCatalogFor(tc.path))
				if err != nil {
					t.Fatal(err)
				}
				n := 0
				for _, f := range findings {
					if f.Rule == "zellij-native-admission" {
						n++
					}
				}
				return n
			}
			if scan(original) != 0 {
				t.Fatal("production source is not admitted")
			}
			mutation := []byte(strings.Replace(string(original), tc.old, tc.replacement, 1))
			if scan(mutation) == 0 {
				t.Fatal("admission bypass was not detected")
			}
		})
	}
}

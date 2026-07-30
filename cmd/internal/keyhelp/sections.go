package keyhelp

import (
	"fmt"
	"sort"
	"strings"

	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

// Sections builds the help document: wording from each row's named source,
// grouping and order from the catalog.
//
// There is deliberately NO "whichever source has prose wins" fallback. A row whose
// named source has no wording is an error, not an occasion to borrow a sentence from
// somewhere else — that fallback is precisely what would render Alt+t as
// "right-terminal tab helper disabled in draft" (its DRAFT no-op desc) instead of
// "new terminal tab" (#132).
func Sections(src SourceReader) ([]Section, error) {
	lua, err := src.Read(nvimInitPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", nvimInitPath, err)
	}
	kdl, err := src.Read(zellijConfigPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", zellijConfigPath, err)
	}

	nvimDesc := map[string]string{}
	scan := ParseNvimKeymaps(string(lua))
	for _, km := range scan.Resolved {
		// First occurrence wins: <M-BS> is mapped twice (normal vs insert mode) with
		// mode-specific wording, and the help describes the action, not the mode.
		if _, seen := nvimDesc[km.Key]; !seen {
			nvimDesc[km.Key] = km.Desc
		}
	}
	for _, km := range scan.Dynamic {
		if _, seen := nvimDesc[km.Raw]; !seen {
			nvimDesc[km.Raw] = km.Desc
		}
	}

	globalHelp := map[string]string{}
	for _, b := range workbenchshortcut.GlobalBindings() {
		globalHelp[b.NvimKey] = b.Help
	}
	roleHelp := map[string]string{}
	for _, rb := range workbenchshortcut.RoleBindings() {
		if k := roleChordKey(rb.Chord); k != "" {
			roleHelp[k] = rb.Help
		}
	}
	zellijLive := map[string]bool{}
	for _, z := range ParseZellijRunBinds(string(kdl)) {
		zellijLive[z.Key] = true
	}

	byGroup := map[string][]Binding{}
	for _, e := range Catalog.include {
		desc, err := descFor(e, nvimDesc, globalHelp, roleHelp, zellijLive)
		if err != nil {
			return nil, err
		}
		byGroup[e.Group] = append(byGroup[e.Group], Binding{
			Key:     displayFor(e),
			Desc:    desc,
			Context: e.Context,
			Group:   e.Group,
			Order:   e.Order,
		})
	}

	var out []Section
	for _, g := range groupOrder {
		rows := byGroup[g]
		if len(rows) == 0 {
			continue
		}
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].Order < rows[j].Order })
		out = append(out, Section{Title: g, Bindings: rows})
	}
	return out, nil
}

// descFor resolves a row's wording from the source it names — and only that source.
func descFor(e entry, nvimDesc, globalHelp, roleHelp map[string]string, zellijLive map[string]bool) (string, error) {
	switch e.Source {
	case SourceNvim:
		if d := nvimDesc[e.Key]; d != "" {
			return d, nil
		}
		return "", fmt.Errorf("keyhelp: %q names SourceNvim but nvim/init.lua has no `pair:` desc for it", e.Key)
	case SourceGlobal:
		if d := globalHelp[e.Key]; d != "" {
			return d, nil
		}
		return "", fmt.Errorf("keyhelp: %q names SourceGlobal but no GlobalBinding.Help matches", e.Key)
	case SourceRole:
		if d := roleHelp[e.Key]; d != "" {
			return d, nil
		}
		return "", fmt.Errorf("keyhelp: %q names SourceRole but no RoleBinding.Help matches", e.Key)
	case SourceZellij:
		// The only rows whose wording is authored in the catalog: zellij's KDL has no
		// description field, so there is no upstream prose to derive. Still verify the
		// bind exists, so a removed bind cannot linger as help.
		if !zellijLive[e.Key] {
			return "", fmt.Errorf("keyhelp: %q names SourceZellij but config.kdl has no such Run bind", e.Key)
		}
		if strings.TrimSpace(e.Help) == "" {
			return "", fmt.Errorf("keyhelp: %q is SourceZellij and must carry Help in the catalog", e.Key)
		}
		return e.Help, nil
	}
	return "", fmt.Errorf("keyhelp: %q has no Source", e.Key)
}

// displayFor returns the row's display spelling, following PAIR_CHEATS' convention
// (Alt+⏎, Alt+x) so the statusline cheatsheet and this help agree.
func displayFor(e entry) string {
	if e.Display != "" {
		return e.Display
	}
	return e.Key
}

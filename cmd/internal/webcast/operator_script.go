package webcast

import (
	"fmt"
	"regexp"
	"strings"
)

// The operator uses the viewers' actual modules in Chrome's isolated world,
// where Page.addScriptToEvaluateOnNewDocument takes a classic script. This
// adapter accepts only our embedded modules' named declarations/imports. It
// is deliberately not a JavaScript transpiler; unsupported module syntax
// fails assembly instead of silently shipping a divergent copy of the UI.
var (
	operatorExport = regexp.MustCompile(`^export ((?:async )?function|const) ([A-Za-z_$][A-Za-z0-9_$]*)\b`)
	operatorImport = regexp.MustCompile(`^import \{ ([A-Za-z0-9_$, ]+) \} from '([^']+)';$`)
	operatorName   = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)
)

func assembleOperatorModule(source string, imports map[string]string) (string, error) {
	var names []string
	lines := strings.Split(source, "\n")
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trim, "import "):
			m := operatorImport.FindStringSubmatch(trim)
			if m == nil || !operatorName.MatchString(imports[m[2]]) {
				return "", fmt.Errorf("operator script: unsupported import on line %d", i+1)
			}
			for _, name := range strings.Split(m[1], ",") {
				if !operatorName.MatchString(strings.TrimSpace(name)) {
					return "", fmt.Errorf("operator script: unsupported import name on line %d", i+1)
				}
			}
			lines[i] = "const { " + m[1] + " } = " + imports[m[2]] + ";"
		case strings.HasPrefix(trim, "export "):
			m := operatorExport.FindStringSubmatch(trim)
			if m == nil {
				return "", fmt.Errorf("operator script: unsupported export on line %d", i+1)
			}
			names = append(names, m[2])
			lines[i] = strings.TrimPrefix(trim, "export ")
		}
	}
	return "(() => {\n" + strings.Join(lines, "\n") + "\nreturn {" + strings.Join(names, ",") + "};\n})()", nil
}

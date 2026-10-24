package webcast

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/broadcast"
	"regexp"
	"strings"
)

// The operator uses the viewers' actual modules in Chrome's isolated world,
// where Page.addScriptToEvaluateOnNewDocument takes a classic script. This
// adapter accepts only our embedded modules' named declarations/imports. It
// is deliberately not a JavaScript transpiler; unsupported module syntax
// fails assembly instead of silently shipping a divergent copy of the UI.
var (
	operatorExportNames = regexp.MustCompile(`^export \{ ([A-Za-z0-9_$, ]+) \};$`)
	operatorExport      = regexp.MustCompile(`^export ((?:async )?function|const|class) ([A-Za-z_$][A-Za-z0-9_$]*)\b`)
	operatorImport      = regexp.MustCompile(`^import \{ ([A-Za-z0-9_$, ]+) \} from '([^']+)';$`)
	operatorName        = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)
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
			if m := operatorExportNames.FindStringSubmatch(trim); m != nil {
				for _, name := range strings.Split(m[1], ",") {
					name = strings.TrimSpace(name)
					if !operatorName.MatchString(name) {
						return "", fmt.Errorf("operator script: unsupported export name on line %d", i+1)
					}
					names = append(names, name)
				}
				lines[i] = ""
				continue
			}
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

//go:embed operator.js
var operatorJS string

//go:embed operator-sketch.js
var operatorSketchJS string

//go:embed operator-policy.js
var suppressionJS string

func buildOperatorScript() string {
	module := func(source string, imports map[string]string) string {
		result, err := assembleOperatorModule(source, imports)
		if err != nil {
			panic(err)
		}
		return result
	}
	read := func(name string) string {
		b, err := webFS.ReadFile("web/" + name)
		if err != nil {
			panic(err)
		}
		return string(b)
	}
	palette, _ := json.Marshal(broadcast.PointerPalette())
	css, _ := json.Marshal(read("annotate.css") + "\n" + read("feedback-panel.css") + "\n" + read("cast-tools.css"))
	sketchCSS, _ := json.Marshal(read("sketch.css"))
	shellURL, _ := json.Marshal(SketchSceneShellURL())
	sketchModules := ";const sketchPolicy=" + module(read("sketch-policy.js"), nil) + ";const sketchRender=" + module(read("sketch-render.js"), map[string]string{"./annotation-anchor.js": "anchor"}) + ";const sketchDelivery=" + module(read("sketch-delivery.js"), nil) + ";const sketchNotes=" + module(read("sketch-notes.js"), nil) + ";const sketchUI=" + module(read("sketch.js"), map[string]string{"./sketch-policy.js": "sketchPolicy", "./sketch-render.js": "sketchRender", "./sketch-notes.js": "sketchNotes", "./feedback-panel.js": "feedbackPanel", "./feedback-glow.js": "feedbackGlow"}) + ";const operatorSketch=" + module(operatorSketchJS, map[string]string{"./web/sketch.js": "sketchUI", "./web/sketch-render.js": "sketchRender", "./web/sketch-delivery.js": "sketchDelivery"})
	return "(()=>{const feedbackMask=" + module(read("feedback-mask.js"), nil) + ";const collab=" + module(string(broadcast.CollabJS()), nil) + ";const policy=" + module(read("annotation-policy.js"), nil) + ";const anchor=" + module(read("annotation-anchor.js"), nil) + ";const feedbackVisibility=" + module(read("feedback-visibility.js"), map[string]string{"./annotation-anchor.js": "anchor"}) + ";const placement=" + module(read("annotation-placement.js"), map[string]string{"./annotation-anchor.js": "anchor"}) + ";const capture=" + module(read("capture-draft.js"), nil) + ";const floating=" + module(read("floating-panel.js"), nil) + ";const feedbackEntriesModule=" + module(read("feedback-entries.js"), nil) + ";const feedbackLayoutModule=" + module(read("feedback-layout.js"), nil) + ";const feedbackGlow=" + module(read("feedback-glow.js"), nil) + ";const feedbackPanel=" + module(read("feedback-panel.js"), map[string]string{"./feedback-entries.js": "feedbackEntriesModule", "./floating-panel.js": "floating"}) + ";const castTools=" + module(read("cast-tools.js"), map[string]string{"./feedback-layout.js": "feedbackLayoutModule"}) + ";const annotate=" + module(read("annotate.js"), map[string]string{"./collab.js": "collab", "./annotation-policy.js": "policy", "./capture-draft.js": "capture", "./feedback-panel.js": "feedbackPanel", "./feedback-glow.js": "feedbackGlow"}) + sketchModules + ";const operator=" + module(operatorJS, map[string]string{"./web/annotation-placement.js": "placement", "./web/feedback-mask.js": "feedbackMask", "./web/feedback-visibility.js": "feedbackVisibility", "./web/feedback-layout.js": "feedbackLayoutModule", "./web/cast-tools.js": "castTools"}) + ";const suppression=" + module(suppressionJS, nil) + ";globalThis.__couchcastSuppressionPolicy=suppression.suppressionPolicy;globalThis.__couchcastMountOperator=(parts)=>operator.mountOperator(parts,{annotate,palette:" + string(palette) + ",css:" + string(css) + ",sketch:{mount:operatorSketch.mountOperatorSketch,css:" + string(sketchCSS) + ",shellURL:" + string(shellURL) + "}});})();\n" + controlJS
}

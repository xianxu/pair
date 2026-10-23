package sessionsync

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xianxu/pair/cmd/internal/editorial"
)

// The immutable embedded catalog is shared by all transaction snapshots.
var sharedEditorialCatalog, sharedEditorialCatalogError = editorial.DefaultCatalog()

func compileEditorial(content string) (*editorial.DocumentPolicy, error) {
	if sharedEditorialCatalogError != nil {
		return nil, sharedEditorialCatalogError
	}
	p, err := editorial.Compile([]byte(content), sharedEditorialCatalog)
	if err != nil {
		return nil, &Refusal{Exit: ExitRefused, Code: "editorial-invalid", Reason: err.Error()}
	}
	return p, nil
}

// editorialPolicy belongs to this transaction's single loaded source snapshot.
func (f *file) editorialPolicy() (*editorial.DocumentPolicy, error) {
	if f.policy != nil {
		return f.policy, nil
	}
	p, err := compileEditorial(f.content)
	if err == nil {
		f.policy = p
	}
	return p, err
}

// sectionPolicyHash binds section bytes to document settings and the heading's
// ancestor authority. Unrelated prose does not invalidate its opaque token.
func sectionPolicyHash(content, heading string, p *editorial.DocumentPolicy) (string, error) {
	body, err := SectionBody(content, heading)
	if err != nil {
		return "", err
	}
	lines := strings.Split(content, "\n")
	i, _, err := section(lines, heading)
	if err != nil {
		return "", err
	}
	start := 0
	for _, line := range lines[:i] {
		start += len(line) + 1
	}
	selected := -1
	for n, h := range p.Headings {
		if h.Range.Start == start {
			selected = n
			break
		}
	}
	if selected < 0 {
		return "", fmt.Errorf("section %q is not an editorial ATX heading", heading)
	}
	type context struct {
		Name      string
		Level     int
		Directive string
	}
	var ancestors []context
	for at := selected; at >= 0; at = p.Headings[at].Parent {
		h := p.Headings[at]
		x := context{Name: h.Name, Level: h.Level}
		if h.Directive != nil {
			x.Directive = string(h.Directive.Raw)
		}
		ancestors = append(ancestors, x)
	}
	var document []string
	for _, owned := range p.Owned {
		if owned.Heading < 0 {
			document = append(document, string(owned.Raw))
		}
	}
	raw, err := json.Marshal(struct {
		Domain, Body string
		Document     []string
		Ancestors    []context
	}{"pair/editorial-section/v1", body, document, ancestors})
	if err != nil {
		return "", err
	}
	return Hash(string(raw)), nil
}

func targetPolicyHash(content, section string, frontmatter bool, p *editorial.DocumentPolicy) (string, error) {
	if frontmatter {
		block, err := FrontmatterBody(content)
		return Hash(block), err
	}
	if section != "" {
		return sectionPolicyHash(content, section, p)
	}
	return Hash(content), nil
}

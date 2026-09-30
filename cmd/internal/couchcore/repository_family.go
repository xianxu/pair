package couchcore

import (
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// MaxRepositoryFamilies bounds permanently retained family descriptors per store.
const MaxRepositoryFamilies = 4096

// RepositoryFamily retains one starting directory across a Git repository's checkouts.
type RepositoryFamily struct {
	RepoIdentity  string `json:"repo_identity"`
	PrimaryRoot   string `json:"primary_root"`
	RelativeStart string `json:"relative_start"`
}

func (f RepositoryFamily) Validate() error {
	if !workspaceAbsolute(f.RepoIdentity) || !workspaceAbsolute(f.PrimaryRoot) {
		return errors.New("repository family requires canonical absolute identity and primary root")
	}
	_, err := ProjectFamilyPath(f.PrimaryRoot, f.RelativeStart)
	return err
}

// ResolveFamilyStart uses an empty requested directory to inherit. A literal dot
// is an explicit root request and conflicts with a retained subdirectory.
func ResolveFamilyStart(saved *RepositoryFamily, requested RepositoryFamily) (RepositoryFamily, error) {
	if !workspaceAbsolute(requested.RepoIdentity) || !workspaceAbsolute(requested.PrimaryRoot) {
		return RepositoryFamily{}, errors.New("invalid requested repository family")
	}
	if requested.RelativeStart != "" {
		if err := requested.Validate(); err != nil {
			return RepositoryFamily{}, err
		}
	}
	if saved == nil {
		if requested.RelativeStart == "" {
			requested.RelativeStart = "."
		}
		return requested, requested.Validate()
	}
	if err := saved.Validate(); err != nil {
		return RepositoryFamily{}, err
	}
	if requested.RepoIdentity != saved.RepoIdentity || requested.PrimaryRoot != saved.PrimaryRoot {
		return RepositoryFamily{}, fmt.Errorf("repository family identity conflicts with retained family %s (%s)", saved.PrimaryRoot, saved.RepoIdentity)
	}
	if requested.RelativeStart != "" && requested.RelativeStart != saved.RelativeStart {
		return RepositoryFamily{}, fmt.Errorf("repository family %s already starts at %s; requested %s conflicts with retained directory %s", saved.PrimaryRoot, filepath.Join(saved.PrimaryRoot, saved.RelativeStart), filepath.Join(requested.PrimaryRoot, requested.RelativeStart), saved.RelativeStart)
	}
	return *saved, nil
}

// ProjectFamilyPath is lexical and permits a checkout that has not been created yet.
func ProjectFamilyPath(root, relative string) (string, error) {
	if !workspaceAbsolute(root) {
		return "", fmt.Errorf("invalid checkout root %q", root)
	}
	if relative == "" || !utf8.ValidString(relative) || strings.ContainsAny(relative, "\\\x00") || filepath.IsAbs(relative) || filepath.Clean(relative) != relative {
		return "", fmt.Errorf("invalid family starting directory %q", relative)
	}
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		if part == ".." {
			return "", fmt.Errorf("family starting directory escapes checkout: %q", relative)
		}
	}
	return filepath.Join(root, relative), nil
}

// RelativeFamilyPath compares canonical snapshots without requiring historical
// directories to remain on disk. Action admission performs physical validation.
func RelativeFamilyPath(root, path string) (string, error) {
	if !workspaceAbsolute(root) || !workspaceAbsolute(path) {
		return "", errors.New("checkout membership requires canonical absolute paths")
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return "", err
	}
	projected, err := ProjectFamilyPath(root, relative)
	if err != nil || projected != path {
		return "", fmt.Errorf("path %s is outside checkout %s", path, root)
	}
	return relative, nil
}

// CheckoutMembership requires repository authority as well as path containment.
// A common directory alone can span several checkouts; an exact scope names one.
func CheckoutMembership(commonGit, checkoutRoot, path, scope, observedCommonGit string) (string, bool, error) {
	expected, err := launcher.ResolveRepoScope(checkoutRoot)
	if err != nil {
		return "", false, err
	}
	if scope != "" && scope != expected.Key || observedCommonGit != "" && observedCommonGit != commonGit || scope == "" && observedCommonGit == "" {
		return "", false, nil
	}
	relative, err := RelativeFamilyPath(checkoutRoot, path)
	if err != nil {
		if scope == "" {
			return "", false, nil
		}
		return "", false, err
	}
	return relative, true, nil
}

// RecordCheckoutMembership shares the storage and inference ownership rule.
func RecordCheckoutMembership(record ThreadRecord, commonGit, checkoutRoot string) (string, bool, error) {
	observed := ""
	for _, incarnation := range record.Incarnations {
		if incarnation.RepoIdentity != "" {
			if incarnation.RepoIdentity != commonGit {
				return "", false, nil
			}
			observed = incarnation.RepoIdentity
		}
	}
	return CheckoutMembership(commonGit, checkoutRoot, record.StartingPath, record.Address.RepoScope, observed)
}

// ValidateFamilyPath resolves physical aliases and requires an existing directory
// inside the actual checkout, catching symlink escapes before a child can spawn.
func ValidateFamilyPath(root, relative string) (string, error) {
	path, err := ProjectFamilyPath(root, relative)
	if err != nil {
		return "", err
	}
	physicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("family checkout %s: %w", root, err)
	}
	physical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("configured family starting directory %s: %w", path, err)
	}
	if _, err = RelativeFamilyPath(physicalRoot, physical); err != nil {
		return "", fmt.Errorf("configured family starting directory %s escapes checkout %s", path, root)
	}
	info, err := os.Stat(physical)
	if err != nil {
		return "", fmt.Errorf("configured family starting directory %s: %w", path, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("configured family starting directory %s is not a directory", path)
	}
	return physical, nil
}

// InferRepositoryFamily considers readable retained paths from every checkout.
// It never chooses arbitrarily between inconsistent legacy starting directories.
func InferRepositoryFamily(repository SlotRepository, records []ThreadRecord) (RepositoryFamily, bool, error) {
	family := RepositoryFamily{RepoIdentity: repository.Identity.RepoIdentity, PrimaryRoot: repository.Identity.PrimaryRoot}
	roots := []string{family.PrimaryRoot}
	for _, slot := range repository.Slots {
		roots = append(roots, slot.Identity.WorktreeRoot)
	}
	relativePaths := map[string][]string{}
	for _, record := range records {
		sameRepository := false
		for _, incarnation := range record.Incarnations {
			if incarnation.RepoIdentity == family.RepoIdentity {
				sameRepository = true
			}
		}
		matched := false
		for _, root := range roots {
			relative, belongs, err := RecordCheckoutMembership(record, family.RepoIdentity, root)
			if err != nil {
				return RepositoryFamily{}, false, fmt.Errorf("repository family %s has retained path %s outside its recorded checkout %s; resolve its checkout identity before creating another conversation: %w", family.PrimaryRoot, record.StartingPath, root, err)
			}
			if !belongs {
				continue
			}
			relativePaths[relative] = append(relativePaths[relative], record.StartingPath)
			matched = true
			break
		}
		if sameRepository && !matched {
			return RepositoryFamily{}, false, fmt.Errorf("repository family %s has retained starting path %s in an unprojectable checkout; resolve its checkout identity before creating another conversation", family.PrimaryRoot, record.StartingPath)
		}
	}
	if len(relativePaths) == 0 {
		return family, false, nil
	}
	if len(relativePaths) > 1 {
		var paths []string
		for _, values := range relativePaths {
			paths = append(paths, values...)
		}
		sort.Strings(paths)
		return RepositoryFamily{}, false, fmt.Errorf("repository family %s has conflicting retained starting paths: %s; preserve these conversations and resolve their family directory before creating another", family.PrimaryRoot, strings.Join(paths, ", "))
	}
	for relative := range relativePaths {
		family.RelativeStart = relative
	}
	return family, true, family.Validate()
}

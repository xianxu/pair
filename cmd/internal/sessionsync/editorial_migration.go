package sessionsync

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"

	"github.com/xianxu/pair/cmd/internal/editorial"
)

// EditorialMigrationRequest accepts no replacement payload. Confirmation names
// the exact source whose compiler-produced candidate the operator previewed.
type EditorialMigrationRequest struct {
	File    string
	Confirm bool
	IfHash  string
}

// EditorialMigrationResult carries the exact candidate as a JSON string, so a
// preview is inspectable without any document or acknowledgement mutation.
type EditorialMigrationResult struct {
	BeforeHash  string                 `json:"beforeHash"`
	Candidate   string                 `json:"candidate"`
	Hash        string                 `json:"hash"`
	Applied     bool                   `json:"applied"`
	Diagnostics []editorial.Diagnostic `json:"diagnostics,omitempty"`
	Warning     string                 `json:"warning,omitempty"`
}

func migrationHash(source string) string {
	h := sha256.Sum256([]byte(source))
	return hex.EncodeToString(h[:])
}

// MigrateEditorial is an operator transform, rather than an agent write. It
// recomputes the sole permitted transform under the ordinary document lock and
// leaves shadow/state untouched so agents still see the operator's change.
func (s *Sync) MigrateEditorial(req EditorialMigrationRequest) (EditorialMigrationResult, error) {
	if req.Confirm {
		raw, err := hex.DecodeString(req.IfHash)
		if err != nil || len(raw) != sha256.Size || hex.EncodeToString(raw) != req.IfHash {
			return EditorialMigrationResult{}, refuse("confirmation requires the full lowercase SHA256 --if-hash from migration preview")
		}
	} else if req.IfHash != "" {
		return EditorialMigrationResult{}, refuse("--if-hash requires --confirm")
	}
	return withDocument(s, req.File, func(f *file) (EditorialMigrationResult, error) {
		if err := f.guard(s.now()); err != nil {
			return EditorialMigrationResult{}, err
		}
		before := migrationHash(f.content)
		if req.Confirm && req.IfHash != before {
			return EditorialMigrationResult{}, &Refusal{Exit: ExitStale, Code: "stale", Hash: before, Reason: "the source changed since migration preview; preview again before confirming"}
		}
		if sharedEditorialCatalogError != nil {
			return EditorialMigrationResult{}, sharedEditorialCatalogError
		}
		plan, err := editorial.PlanMigration([]byte(f.content), sharedEditorialCatalog)
		if err != nil {
			return EditorialMigrationResult{}, &Refusal{Exit: ExitRefused, Code: "editorial-invalid", Reason: err.Error()}
		}
		after := string(plan.After)
		if f.registration != nil && len(plan.After) > documentMaxBytes {
			return EditorialMigrationResult{}, registryLimit()
		}
		// Defend the operator transform's authority boundary independently of the
		// compiler: registration/type/routing ownership must never be rewritten.
		owner := ParseOwner(after)
		if f.registration != nil {
			owner = EffectiveOwner(after, f.registration.RegisteredOwner)
		}
		if plan.BeforeHash != before || owner != f.owner || frontmatter(after, "type") != frontmatter(f.content, "type") {
			return EditorialMigrationResult{}, refuse("editorial migration would change document identity or owner")
		}
		result := EditorialMigrationResult{BeforeHash: before, Candidate: after, Hash: migrationHash(after), Diagnostics: plan.Diagnostics}
		if !req.Confirm || after == f.content {
			return result, nil
		}
		// Clearing before publication prevents an old round being rendered over new
		// coordinates, including when publication's durability cannot be confirmed.
		if err := os.Remove(f.paths.Round); err != nil && !errors.Is(err, os.ErrNotExist) {
			return EditorialMigrationResult{}, refuse("the previous round could not be cleared: %v", err)
		}
		if err := replaceShared(f.paths.File, plan.After); err != nil {
			return EditorialMigrationResult{}, &Refusal{Exit: ExitRefused, Code: "migration-uncertain", Reason: fmt.Sprintf("migration publication could not be confirmed: %v; inspect the document and preview again", err)}
		}
		result.Applied = true
		lockPaths := f.lockPaths
		if len(lockPaths) == 0 {
			lockPaths = []string{f.paths.Lock}
		}
		for _, path := range lockPaths {
			lock, exists, err := readOptional(path)
			if err != nil {
				result.Warning = "the operator lock could not be inspected after migration; check the editor before continuing"
				break
			}
			if ParseHolder(lock, exists) == HolderOperator {
				result.Warning = "the operator took the lock during migration; their buffer reloads with it"
				break
			}
		}
		return result, nil
	})
}

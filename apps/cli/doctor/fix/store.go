package fix

import (
	"context"

	"github.com/borg0ai/dx-issuegraph/internal/eventsjournal"
	"github.com/borg0ai/dx-issuegraph/internal/storage/dolt"
)

// openIssueMutatingStore opens a repair handler's store WITH the workspace's
// events-journal activation applied.
//
// Most of issuegraph doctor inspects and repairs workspace state — schema cursors,
// metadata, remotes, fingerprints, git — and none of that belongs in a journal
// whose records a consumer replays onto issuegraph. Three repair handlers are not
// like that: StaleClosedIssues and PatrolPollution DELETE issues, and the two
// fresh-clone import paths CREATE them. Those are ordinary issue mutations, and
// a consumer whose mirror silently diverges because someone ran
// `issuegraph doctor --fix` is exactly the failure the journal exists to prevent — the
// more so because a repair is unattended and nobody is watching the diff.
//
// The read-only and workspace-state handlers deliberately keep opening the
// store directly; they carry their exemption in the construction guard, with a
// reason that is true of them.
// The concrete *dolt.DoltStore is returned rather than storage.DoltStorage
// because these handlers reach past the interface for raw diagnostic SQL, which
// is what a repair path legitimately does.
func openIssueMutatingStore(ctx context.Context, issuegraphDir string) (*dolt.DoltStore, error) {
	store, err := dolt.NewFromConfig(ctx, issuegraphDir)
	return activated(issuegraphDir, store, err)
}

// openIssueMutatingStoreCreating is openIssueMutatingStore for the fresh-clone
// repair, which must be allowed to create the database it is about to import
// into — that is the whole point of the fix, so it opts out of the create
// guard rather than failing on an absent database.
func openIssueMutatingStoreCreating(ctx context.Context, issuegraphDir string) (*dolt.DoltStore, error) {
	store, err := dolt.NewFromConfigWithOptions(ctx, issuegraphDir, &dolt.Config{CreateIfMissing: true})
	return activated(issuegraphDir, store, err)
}

// activated applies the workspace's journal setting and closes the store if it
// cannot be honored. A failed open passes straight through, so the typed-nil
// store never reaches the activation.
func activated(issuegraphDir string, store *dolt.DoltStore, err error) (*dolt.DoltStore, error) {
	if _, err = eventsjournal.ActivateStore(issuegraphDir, store, err); err != nil {
		return nil, err
	}
	return store, nil
}

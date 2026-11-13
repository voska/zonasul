package command

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/voska/vtexkit/cli"
	"github.com/voska/vtexkit/cli/errfmt"
	"github.com/voska/zonasul/internal/bff"
)

// Claim the order before any checkout write. Keep the record on every failure:
// a later invocation cannot establish whether a previous payment reached the
// gateway. O_EXCL also prevents concurrent processes from submitting twice.
func (r *Runner) recordPaymentAttempt(g *cli.Globals, orderID string) error {
	stateDir := r.stateDir
	if stateDir == "" {
		stateDir = g.Config().Dir()
	}
	dir := filepath.Join(stateDir, "payment-attempts")
	failure := func() error {
		return errfmt.Config("could not persist checkout attempt; payment was not sent; inspect orders " + orderID)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return failure()
	}
	// Hash the server-provided ID so it cannot introduce path components.
	name := fmt.Sprintf("%x", sha256.Sum256([]byte(orderID)))
	f, err := os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if os.IsExist(err) {
		return errfmt.New(bff.ExitSubmittedUnverified, "checkout attempt already recorded; do not retry checkout; inspect orders "+orderID)
	}
	if err != nil {
		return failure()
	}
	_, writeErr := f.WriteString(orderID + "\n")
	syncErr := f.Sync()
	closeErr := f.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil {
		return failure()
	}
	return nil
}

package command

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/voska/vtexkit/cli/errfmt"
)

func TestConcurrentCheckoutAttemptsHaveOneWinner(t *testing.T) {
	dir := t.TempDir()
	const callers = 12
	results := make(chan error, callers)
	for range callers {
		go func() {
			results <- (&Runner{stateDir: dir}).recordPaymentAttempt(globals(), "../synthetic-order")
		}()
	}
	winners := 0
	for range callers {
		err := <-results
		if err == nil {
			winners++
			continue
		}
		var typed *errfmt.Error
		if !errors.As(err, &typed) || typed.Code != 11 {
			t.Errorf("concurrent attempt: %v", err)
		}
	}
	if winners != 1 {
		t.Fatalf("successful attempts=%d", winners)
	}
	entries, err := os.ReadDir(filepath.Join(dir, "payment-attempts"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("records=%v err=%v", entries, err)
	}
	record, err := os.ReadFile(filepath.Join(dir, "payment-attempts", entries[0].Name()))
	if err != nil || string(record) != "../synthetic-order\n" {
		t.Fatalf("record=%q err=%v", record, err)
	}
	if err := (&Runner{stateDir: dir}).recordPaymentAttempt(globals(), "another-order"); err != nil {
		t.Fatalf("a different order was blocked: %v", err)
	}
}

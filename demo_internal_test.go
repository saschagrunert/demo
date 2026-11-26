package demo

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/urfave/cli/v3"
)

var errTestCleanup = errors.New("cleanup failed")

func TestRunWithHooksReportsCleanupFailureAfterCtrlC(t *testing.T) {
	t.Parallel()

	d := New()
	d.Cleanup(func(context.Context, *cli.Command) error { return errTestCleanup })

	err := d.runWithHooks(t.Context(), d.Command, func(context.Context, *cli.Command) error {
		return fmt.Errorf("unable to run step: %w", errInterrupt)
	})

	if !errors.Is(err, errTestCleanup) {
		t.Fatalf("expected the cleanup error, got %v", err)
	}

	if errors.Is(err, errInterrupt) {
		t.Fatalf("expected the interruption to be dropped, got %v", err)
	}
}

func TestRunWithHooksReturnsInterruptAfterCtrlC(t *testing.T) {
	t.Parallel()

	d := New()

	err := d.runWithHooks(t.Context(), d.Command, func(context.Context, *cli.Command) error {
		return fmt.Errorf("unable to run step: %w", errInterrupt)
	})

	if !errors.Is(err, errInterrupt) {
		t.Fatalf("expected the interruption, got %v", err)
	}
}

func TestWithoutInterruptionKeepsCleanupErrors(t *testing.T) {
	t.Parallel()

	err := errors.Join(
		fmt.Errorf("step interrupted: %w", context.DeadlineExceeded),
		&cleanupError{err: fmt.Errorf("timed out: %w", context.DeadlineExceeded)},
	)

	got := withoutInterruption(err, context.DeadlineExceeded)

	if _, ok := errors.AsType[*cleanupError](got); !ok {
		t.Fatalf("expected the cleanup error to be kept, got %v", got)
	}
}

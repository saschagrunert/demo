package demo

import (
	"context"
	"fmt"
	"os/exec"
)

// EnsureWithContext executes the provided commands in order with the given context.
// This utility function can be used during setup or cleanup.
func EnsureWithContext(ctx context.Context, commands ...string) error {
	for _, c := range commands {
		cmd := exec.CommandContext(ctx, "sh", "-c", c)
		cmd.Stderr = nil
		cmd.Stdout = nil

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("run command: %w", err)
		}
	}

	return nil
}

// Ensure executes the provided commands in order.
// This utility function can be used during setup or cleanup.
func Ensure(commands ...string) error {
	return EnsureWithContext(context.Background(), commands...)
}

// MustEnsure executes the provided commands in order and panics on failure.
func MustEnsure(commands ...string) {
	if err := Ensure(commands...); err != nil {
		panic(err)
	}
}

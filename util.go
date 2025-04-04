package demo

import (
	"fmt"
	"os/exec"
)

// Ensure executes the provided commands in order.
// This utility function can be used during setup or cleanup.
func Ensure(commands ...string) error {
	for _, c := range commands {
		cmd := exec.Command("sh", "-c", c)
		cmd.Stderr = nil
		cmd.Stdout = nil

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("run command: %w", err)
		}
	}

	return nil
}

// MustEnsure executes the provided commands in order and panics on failure.
func MustEnsure(commands ...string) {
	if err := Ensure(commands...); err != nil {
		panic(err)
	}
}

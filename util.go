package demo

import (
	"os/exec"
)

// Ensure executes the provided commands in order
func Ensure(commands ...string) error {
	for _, c := range commands {
		cmd := exec.Command(bash, "-c", c)
		cmd.Stderr = nil
		cmd.Stdout = nil
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

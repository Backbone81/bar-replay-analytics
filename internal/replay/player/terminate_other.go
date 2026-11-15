//go:build !windows

package player

import (
	"os"
	"syscall"
)

// terminate asks the engine process to quit.
func terminate(process *os.Process) error {
	return process.Signal(syscall.SIGTERM)
}

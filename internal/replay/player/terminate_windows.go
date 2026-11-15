package player

import (
	"os"
)

// terminate stops the engine process. Windows has no SIGTERM and the engine handles no console control event, so
// asking it to quit would end it the same way: it is killed.
func terminate(process *os.Process) error {
	return process.Kill()
}

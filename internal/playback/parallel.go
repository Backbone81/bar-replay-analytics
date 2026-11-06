package playback

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// cpusPerPlayback is the CPUs DefaultParallel counts per playback: with DefaultConfig's WorkerThreadCount, a 27-minute
// 1v1 used 1.4 to 1.7 cores, alone and four at a time.
const cpusPerPlayback = 2

// memoryPerPlayback is the memory DefaultParallel counts per playback: the engine used 4.8 GB resident for that
// replay, and larger maps and team games likely need more.
const memoryPerPlayback = 6 * 1024 * 1024 * 1024

// DefaultParallel returns how many playbacks to run at once here: one per cpusPerPlayback CPUs, at most one per
// memoryPerPlayback of available memory, at least one. Available memory is known on Linux only (MemAvailable).
func DefaultParallel() int {
	return ParallelFor(runtime.NumCPU(), availableMemory())
}

// ParallelFor computes DefaultParallel for cpus CPUs and availableMemory bytes; 0 means unknown and is ignored.
func ParallelFor(cpus int, availableMemory int64) int {
	parallel := cpus / cpusPerPlayback
	if availableMemory > 0 {
		parallel = min(parallel, int(availableMemory/memoryPerPlayback))
	}
	return max(1, parallel)
}

// availableMemory returns MemAvailable of /proc/meminfo in bytes, or 0 if it could not be read.
func availableMemory() int64 {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer file.Close() //nolint:errcheck

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// "MemAvailable:   30192345 kB"
		fields := strings.Fields(scanner.Text())
		if len(fields) == 3 && fields[0] == "MemAvailable:" && fields[2] == "kB" {
			kilobytes, err := strconv.ParseInt(fields[1], 10, 64)
			if err != nil {
				return 0
			}
			return kilobytes * 1024
		}
	}
	return 0
}

package playback

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// cpusPerPlayback is the number of CPUs DefaultParallel counts per playback. With the WorkerThreadCount of
// DefaultConfig, a playback used 1.4 to 1.7 cores, measured with a 1v1 replay of 27 minutes alone and four at a time.
const cpusPerPlayback = 2

// memoryPerPlayback is the memory DefaultParallel counts per playback. The engine used 4.8 GB (resident) for that
// replay; larger maps and team games likely need more, so this leaves room.
const memoryPerPlayback = 6 * 1024 * 1024 * 1024

// DefaultParallel returns how many playbacks can run at the same time on this computer: one per cpusPerPlayback
// CPUs, but at most one per memoryPerPlayback of the memory available now, and at least one. The available memory is
// known on Linux only (MemAvailable in /proc/meminfo); elsewhere only the CPUs count.
func DefaultParallel() int {
	return ParallelFor(runtime.NumCPU(), availableMemory())
}

// ParallelFor returns DefaultParallel for cpus CPUs and availableMemory bytes of available memory, which is ignored
// if it is 0 (not known).
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

package utils

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// MemoryAvailable returns MemAvailable of /proc/meminfo in bytes: the kernel's estimate of the memory available for
// new processes without swapping, including reclaimable page cache. It returns 0 if it cannot be read.
func MemoryAvailable() int {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer file.Close() //nolint:errcheck

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// "MemAvailable:   30192345 kB"
		fields := strings.Fields(scanner.Text())
		if len(fields) != 3 || fields[0] != "MemAvailable:" || fields[2] != "kB" {
			continue
		}
		kilobytes, err := strconv.Atoi(fields[1])
		if err != nil {
			return 0
		}
		return kilobytes * 1024
	}
	return 0
}

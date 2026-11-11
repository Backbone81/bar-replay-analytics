package utils

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// memoryStatusEx is MEMORYSTATUSEX of the Windows API.
type memoryStatusEx struct {
	length               uint32
	memoryLoad           uint32
	totalPhys            uint64
	availPhys            uint64
	totalPageFile        uint64
	availPageFile        uint64
	totalVirtual         uint64
	availVirtual         uint64
	availExtendedVirtual uint64
}

var procGlobalMemoryStatusEx = windows.NewLazySystemDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx")

// MemoryAvailable returns ullAvailPhys of GlobalMemoryStatusEx in bytes: the physical memory that is free or can be
// reclaimed without writing to disk (standby list). It returns 0 if the call fails.
func MemoryAvailable() int {
	status := memoryStatusEx{
		length: uint32(unsafe.Sizeof(memoryStatusEx{})),
	}
	if result, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&status))); result == 0 {
		return 0
	}
	return int(status.availPhys)
}

//go:build perf && windows

package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var getProcessMemoryInfo = windows.NewLazySystemDLL("psapi.dll").NewProc("GetProcessMemoryInfo")

// PROCESS_MEMORY_COUNTERS_EX uses pointer-sized SIZE_T fields on both Windows
// architectures. Private bytes and working set include more than the Go heap.
type processMemoryCounters struct {
	Size, PageFaultCount                               uint32
	PeakWorkingSetSize, WorkingSetSize                 uintptr
	QuotaPeakPagedPoolUsage, QuotaPagedPoolUsage       uintptr
	QuotaPeakNonPagedPoolUsage, QuotaNonPagedPoolUsage uintptr
	PagefileUsage, PeakPagefileUsage, PrivateUsage     uintptr
}

func processMemory() map[string]uint64 {
	var counters processMemoryCounters
	counters.Size = uint32(unsafe.Sizeof(counters))
	ok, _, _ := getProcessMemoryInfo.Call(uintptr(windows.CurrentProcess()), uintptr(unsafe.Pointer(&counters)), uintptr(counters.Size))
	if ok == 0 {
		return nil
	}
	return map[string]uint64{
		"working_set_bytes":      uint64(counters.WorkingSetSize),
		"peak_working_set_bytes": uint64(counters.PeakWorkingSetSize),
		"private_bytes":          uint64(counters.PrivateUsage),
	}
}

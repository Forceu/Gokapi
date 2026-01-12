package helper

import (
	"syscall"
	"unsafe"
)

type diskUsage struct {
	freeBytes  int64
	totalBytes int64
	availBytes int64
}

// GetFreeSpace returns the free space in bytes on the given path
func GetFreeSpace(path string) (uint64, error) {
	h := syscall.MustLoadDLL("kernel32.dll")
	c := h.MustFindProc("GetDiskFreeSpaceExW")

	du := &DiskUsage{}

	c.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(volumePath))),
		uintptr(unsafe.Pointer(&du.freeBytes)),
		uintptr(unsafe.Pointer(&du.totalBytes)),
		uintptr(unsafe.Pointer(&du.availBytes)))
	return du.freeBytes, nil
}

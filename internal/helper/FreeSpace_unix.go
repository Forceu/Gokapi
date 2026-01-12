//go:build !js && !wasm

package helper

import "syscall"

// GetFreeSpace returns the free space in bytes on the given path
func GetFreeSpace(path string) (uint64, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return 0, err
	}
	return stat.Bfree * uint64(stat.Bsize), nil
}

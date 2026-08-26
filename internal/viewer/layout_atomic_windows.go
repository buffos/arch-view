//go:build windows

package viewer

import (
	"syscall"
	"unsafe"
)

const (
	layoutMoveFileReplaceExisting = 0x1
	layoutMoveFileWriteThrough    = 0x8
)

func replaceLayoutFile(source, target string) error {
	sourcePointer, err := syscall.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	targetPointer, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	moveFileEx := syscall.NewLazyDLL("kernel32.dll").NewProc("MoveFileExW")
	result, _, callErr := moveFileEx.Call(
		uintptr(unsafe.Pointer(sourcePointer)),
		uintptr(unsafe.Pointer(targetPointer)),
		layoutMoveFileReplaceExisting|layoutMoveFileWriteThrough,
	)
	if result == 0 {
		return callErr
	}
	return nil
}

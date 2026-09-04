//go:build windows

package projectdocument

import (
	"syscall"
	"unsafe"
)

func replaceFile(source, target string) error {
	sourcePointer, err := syscall.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	targetPointer, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	moveFileEx := syscall.NewLazyDLL("kernel32.dll").NewProc("MoveFileExW")
	result, _, callErr := moveFileEx.Call(uintptr(unsafe.Pointer(sourcePointer)), uintptr(unsafe.Pointer(targetPointer)), 0x1|0x8)
	if result == 0 {
		return callErr
	}
	return nil
}

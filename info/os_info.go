package info

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	getComputerNameA = kernel32.NewProc("GetComputerNameA")
	getComputerNameW = kernel32.NewProc("GetComputerNameW")
)

func GetComputerNameA() (string, error) {
	buf := make([]byte, 256)
	size := uint32(len(buf))

	ret, _, err := syscall.SyscallN(
		getComputerNameA.Addr(),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
		0, // Reserved - Must be 0
	)

	if ret == 0 {
		return "", fmt.Errorf("GetComputerNameA failed: %v", err)
	}

	return string(buf[:size]), nil
}

func GetComputerNameW() (string, error) {
	var buf [256]uint16
	var size = uint32(len(buf))

	ret, _, err := syscall.SyscallN(
		getComputerNameW.Addr(),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
		0,
	)

	if ret == 0 {
		return "", err
	}

	return syscall.UTF16ToString(buf[:size]), nil
}

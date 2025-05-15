package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

func main() {
	for {
		OutputProcess()
		time.Sleep(1 * time.Second)
	}
}

func OutputProcess() {
	// Get the current process information
	process := os.Getpid()
	processName, err := os.Executable()
	if err != nil {
		fmt.Println("Error getting process name:", err)
		return
	}

	// Extract just the base name from the full path
	processName = filepath.Base(processName)

	// Get computer name
	computerName, _ := GetComputerName()

	// Output the process name, PID and computer name
	fmt.Printf("Computer: %s | Process: %s (PID: %d)\n", computerName, processName, process)
}

func GetComputerName() (string, error) {
	var buf [256]uint16
	var size uint32 = uint32(len(buf))

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

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	getComputerNameW = kernel32.NewProc("GetComputerNameW")
)

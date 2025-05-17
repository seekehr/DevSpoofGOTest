package main

import (
	"fmt"
	"github.com/fatih/color"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
	"unsafe"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	getComputerNameA = kernel32.NewProc("GetComputerNameA")
	getComputerNameW = kernel32.NewProc("GetComputerNameW")
	getVolumeSerialA = kernel32.NewProc("GetVolumeInformationA")
	getVolumeSerialW = kernel32.NewProc("GetVolumeInformationW")
)

func main() {
	i := 0
	for {
		OutputProcess(i)
		time.Sleep(4 * time.Second)
		i++
	}
}

var red = color.New(color.FgRed).SprintFunc()
var green = color.New(color.FgGreen).SprintFunc()
var blue = color.New(color.FgBlue).SprintFunc()
var cyan = color.New(color.FgCyan).SprintFunc()

func OutputProcess(iteration int) {
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
	computerNameA, errCompA := GetComputerNameA()
	computerNameW, errCompW := GetComputerNameW()

	// Get volume serial using method A
	volumeSerialA, errVolA := GetVolumeSerialA()
	volumeSerialW, errVolW := GetVolumeSerialW()

	// Output the process name, PID and computer name
	fmt.Println("=========" + blue("DevSpoofGOTest.exe "+strconv.Itoa(iteration)) + "=========")
	fmt.Println(green("PID: ") + strconv.Itoa(process))

	pcNameStr := green("PC name: ")
	if errCompA != nil {
		pcNameStr += red("Error getting computer name ("+errCompA.Error()+")") + " || "
	} else {
		pcNameStr += computerNameA + cyan(" || ")
	}
	if errCompW != nil {
		pcNameStr += red("Error getting computer name (" + errCompW.Error() + ")")
	} else {
		pcNameStr += computerNameW
	}
	fmt.Println(pcNameStr)

	volSerialStr := green("Volume Serial: ")
	if errVolA != nil {
		volSerialStr += red("Error getting computer name ("+errVolA.Error()+")") + " || "
	} else {
		volSerialStr += volumeSerialA + cyan(" || ")
	}
	if errVolW != nil {
		volSerialStr += red("Error getting computer name (" + errVolW.Error() + ")")
	} else {
		volSerialStr += volumeSerialW
	}
	fmt.Println(volSerialStr)

	fmt.Println("=====================\n")
}

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

// GetVolumeSerialA retrieves the volume serial number using GetVolumeInformationA,
// requesting only the serial number.
func GetVolumeSerialA() (string, error) {
	drive := "C:\\"
	// Use BytePtrFromString to get a pointer to a null-terminated ANSI string.
	// Note: This works well for basic ASCII characters in paths.
	// For more complex characters, a dedicated ANSI encoding conversion might be needed.
	drivePtr, err := syscall.BytePtrFromString(drive)
	if err != nil {
		return "", fmt.Errorf("failed to convert drive string to byte pointer: %w", err)
	}

	var volumeSerial uint32

	// GetVolumeInformationA function signature (simplified for relevant parameters):
	// BOOL GetVolumeInformationA(
	//   LPCSTR  lpRootPathName,         // Pointer to root directory string (required)
	//   LPSTR   lpVolumeNameBuffer,     // Buffer for volume name (can be NULL)
	//   DWORD   nVolumeNameSize,        // Size of volume name buffer (0 if NULL)
	//   LPDWORD lpVolumeSerialNumber,   // Pointer to volume serial number (can be NULL)
	//   LPDWORD lpMaximumComponentLength, // Pointer to max component length (can be NULL)
	//   LPDWORD lpFileSystemFlags,      // Pointer to file system flags (can be NULL)
	//   LPSTR   lpFileSystemNameBuffer, // Buffer for file system name (can be NULL)
	//   DWORD   nFileSystemNameSize     // Size of file system name buffer (0 if NULL)
	// );

	// We only need the volume serial number. Pass NULL (uintptr(0)) and 0
	// for all other optional parameters to avoid the "More data is available" error
	// if those pieces of information are larger than the provided (or implicitly sized) buffers.
	ret, _, _ := syscall.SyscallN(
		getVolumeSerialA.Addr(),
		uintptr(unsafe.Pointer(drivePtr)),      // lpRootPathName
		uintptr(0),                             // lpVolumeNameBuffer (NULL)
		uintptr(0),                             // nVolumeNameSize (0)
		uintptr(unsafe.Pointer(&volumeSerial)), // lpVolumeSerialNumber
		uintptr(0),                             // lpMaximumComponentLength (NULL)
		uintptr(0),                             // lpFileSystemFlags (NULL)
		uintptr(0),                             // lpFileSystemNameBuffer (NULL)
		uintptr(0),                             // nFileSystemNameSize (0)
	)

	// GetVolumeInformationA returns non-zero on success.
	if ret == 0 {
		// Use GetLastError to get the specific error code from the Windows API.
		// The err returned by SyscallN is sometimes generic, GetLastError is preferred
		// for more detailed error information from the API call itself.
		return "", fmt.Errorf("GetVolumeInformationA failed: %v", syscall.GetLastError())
	}

	// Format the volume serial number as a hexadecimal string.
	return fmt.Sprintf("%X", volumeSerial), nil
}

// GetVolumeSerialW retrieves the volume serial number using GetVolumeInformationW,
// requesting only the serial number. It uses UTF-16 strings for path names.
func GetVolumeSerialW() (string, error) {
	drive := "C:\\"
	// Use UTF16PtrFromString to get a pointer to a null-terminated UTF-16 string.
	drivePtr, err := syscall.UTF16PtrFromString(drive)
	if err != nil {
		return "", fmt.Errorf("failed to convert drive string to UTF16 pointer: %w", err)
	}

	var volumeSerial uint32

	// GetVolumeInformationW function signature (simplified for relevant parameters):
	// BOOL GetVolumeInformationW(
	//   LPCWSTR lpRootPathName,         // Pointer to root directory string (required, UTF-16)
	//   LPWSTR  lpVolumeNameBuffer,     // Buffer for volume name (can be NULL)
	//   DWORD   nVolumeNameSize,        // Size of volume name buffer (0 if NULL)
	//   LPDWORD lpVolumeSerialNumber,   // Pointer to volume serial number (can be NULL)
	//   LPDWORD lpMaximumComponentLength, // Pointer to max component length (can be NULL)
	//   LPDWORD lpFileSystemFlags,      // Pointer to file system flags (can be NULL)
	//   LPWSTR  lpFileSystemNameBuffer, // Buffer for file system name (can be NULL)
	//   DWORD   nFileSystemNameSize     // Size of file system name buffer (0 if NULL)
	// );

	// We only need the volume serial number. Pass NULL (uintptr(0)) and 0
	// for all other optional parameters.
	ret, _, _ := syscall.SyscallN(
		getVolumeSerialW.Addr(),                // Address of GetVolumeInformationW
		uintptr(unsafe.Pointer(drivePtr)),      // lpRootPathName (UTF-16 pointer)
		uintptr(0),                             // lpVolumeNameBuffer (NULL)
		uintptr(0),                             // nVolumeNameSize (0)
		uintptr(unsafe.Pointer(&volumeSerial)), // lpVolumeSerialNumber
		uintptr(0),                             // lpMaximumComponentLength (NULL)
		uintptr(0),                             // lpFileSystemFlags (NULL)
		uintptr(0),                             // lpFileSystemNameBuffer (NULL)
		uintptr(0),                             // nFileSystemNameSize (0)
	)

	// GetVolumeInformationW returns non-zero on success.
	if ret == 0 {
		// Use GetLastError for the specific Windows API error code.
		return "", fmt.Errorf("GetVolumeInformationW failed: %v", syscall.GetLastError())
	}

	// Format the volume serial number as a hexadecimal string.
	return fmt.Sprintf("%X", volumeSerial), nil
}

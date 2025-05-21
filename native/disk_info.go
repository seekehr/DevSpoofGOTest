package native

import (
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var (
	getVolumeSerialA = kernel32.NewProc("GetVolumeInformationA")
	getVolumeSerialW = kernel32.NewProc("GetVolumeInformationW")
)

const (
	IOCTL_STORAGE_QUERY_PROPERTY         = 0x2D1400 // Corrected IOCTL for StorageDeviceProperty
	PropertyStandardQuery                = 0
	StorageDeviceProperty                = 0
	IOCTL_VOLUME_GET_VOLUME_DISK_EXTENTS = 0x560000
)

type StoragePropertyQuery struct {
	PropertyID uint32
	QueryType  uint32
	_          [1]byte // AdditionalParameters
}

type StorageDeviceDescriptor struct {
	Version               uint32
	Size                  uint32
	DeviceType            uint8
	DeviceTypeModifier    uint8
	RemovableMedia        bool
	CommandQueueing       bool
	VendorIDOffset        uint32
	ProductIDOffset       uint32
	ProductRevisionOffset uint32
	SerialNumberOffset    uint32
	BusType               uint16
	RawPropertiesLength   uint32
}

type DiskExtent struct {
	DiskNumber     uint32
	StartingOffset int64
	ExtentLength   int64
}

type VolumeDiskExtents struct {
	NumberOfDiskExtents uint32
	Extents             [1]DiskExtent
}

func getActiveVol() (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	drive := filepath.VolumeName(pwd)
	if drive == "" {
		if len(pwd) >= 2 && pwd[1] == ':' {
			drive = pwd[:2] + `\`
		} else {
			drive = "C:\\"
		}
	} else {
		if strings.HasSuffix(drive, ":") {
			drive += `\`
		}
	}
	fmt.Println("Drive: " + drive)
	return drive, nil
}

// GetVolumeSerialA retrieves the volume serial number using GetVolumeInformationA,
// requesting only the serial number.
func GetVolumeSerialA() (string, error) {
	drive, err := getActiveVol()
	if err != nil {
		return "", fmt.Errorf("failed to get active volume: %w", err)
	}
	drivePtr, err := syscall.BytePtrFromString(drive)
	if err != nil {
		return "", fmt.Errorf("failed to convert drive string to byte pointer: %w", err)
	}

	var volumeSerial uint32

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

	if ret == 0 {
		return "", fmt.Errorf("GetVolumeInformationA failed: %v", syscall.GetLastError())
	}

	// Format the volume serial number as a hexadecimal string.
	return fmt.Sprintf("%X", volumeSerial), nil
}

func GetVolumeSerialW() (string, error) {
	drive := "C:\\"
	// Use UTF16PtrFromString to get a pointer to a null-terminated UTF-16 string.
	drivePtr, err := syscall.UTF16PtrFromString(drive)
	if err != nil {
		return "", fmt.Errorf("failed to convert drive string to UTF16 pointer: %w", err)
	}

	var volumeSerial uint32
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

func getDiskSerialNumberForPath(drivePath string) (string, error) {
	drivePathPtr, err := windows.UTF16PtrFromString(drivePath)
	if err != nil {
		return "", fmt.Errorf("UTF16PtrFromString for drive path '%s' failed: %w", drivePath, err)
	}

	handle, err := windows.CreateFile(
		drivePathPtr,
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		0,
		0,
	)
	if err != nil {
		return "", fmt.Errorf("CreateFile for '%s' failed: %w (ensure admin rights if needed)", drivePath, err)
	}
	defer windows.CloseHandle(handle)

	query := StoragePropertyQuery{
		PropertyID: StorageDeviceProperty,
		QueryType:  PropertyStandardQuery,
	}

	outBufferSize := uint32(unsafe.Sizeof(StorageDeviceDescriptor{}) + 512)
	outBuffer := make([]byte, outBufferSize)
	var bytesReturned uint32

	err = windows.DeviceIoControl(
		handle,
		IOCTL_STORAGE_QUERY_PROPERTY,
		(*byte)(unsafe.Pointer(&query)),
		uint32(unsafe.Sizeof(query)),
		&outBuffer[0],
		outBufferSize,
		&bytesReturned,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("DeviceIoControl IOCTL_STORAGE_QUERY_PROPERTY on '%s' failed: %w", drivePath, err)
	}

	if bytesReturned < uint32(unsafe.Sizeof(StorageDeviceDescriptor{})) {
		return "", fmt.Errorf("DeviceIoControl on '%s' returned insufficient data (%d bytes) for descriptor", drivePath, bytesReturned)
	}

	descriptor := (*StorageDeviceDescriptor)(unsafe.Pointer(&outBuffer[0]))

	if descriptor.SerialNumberOffset == 0 || descriptor.SerialNumberOffset >= bytesReturned {
		return "", fmt.Errorf("serial number offset invalid or out of bounds on '%s' (offset: %d, returned: %d)", drivePath, descriptor.SerialNumberOffset, bytesReturned)
	}

	var serialNumberBytes []byte
	readEnd := descriptor.SerialNumberOffset + (bytesReturned - descriptor.SerialNumberOffset)
	if readEnd > uint32(len(outBuffer)) {
		readEnd = uint32(len(outBuffer))
	}

	for i := descriptor.SerialNumberOffset; i < readEnd; i++ {
		if outBuffer[i] == 0 {
			break
		}
		serialNumberBytes = append(serialNumberBytes, outBuffer[i])
	}

	serialStr := string(serialNumberBytes)
	trimmedSerial := strings.Builder{}
	for _, r := range serialStr {
		if r > 32 && r < 127 {
			trimmedSerial.WriteRune(r)
		}
	}

	resultSerial := trimmedSerial.String()
	if len(resultSerial) == 0 {
		return fmt.Sprintf("N/A (empty or non-printable on %s)", drivePath), nil
	}
	return resultSerial, nil
}

func GetActiveDriveSerialNumber() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("os.Executable failed: %w", err)
	}

	exePathPtr, err := windows.UTF16PtrFromString(exePath)
	if err != nil {
		return "", fmt.Errorf("UTF16PtrFromString for exe path '%s' failed: %w", exePath, err)
	}

	volumePathNameBuf := make([]uint16, windows.MAX_PATH)
	err = windows.GetVolumePathName(exePathPtr, &volumePathNameBuf[0], uint32(len(volumePathNameBuf)))
	if err != nil {
		return "", fmt.Errorf("GetVolumePathName for '%s' failed: %w", exePath, err)
	}
	volumeMountPoint := windows.UTF16ToString(volumePathNameBuf)
	baseVolumePath := strings.TrimSuffix(volumeMountPoint, "\\")
	if len(baseVolumePath) == 0 {
		return "", fmt.Errorf("could not derive base volume path from: '%s'", volumeMountPoint)
	}
	volumeDevicePath := `\\.\` + baseVolumePath

	volPathPtr, err := windows.UTF16PtrFromString(volumeDevicePath)
	if err != nil {
		return "", fmt.Errorf("UTF16PtrFromString for volume device path '%s' failed: %w", volumeDevicePath, err)
	}

	hVolume, err := windows.CreateFile(
		volPathPtr,
		0,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		0,
		0,
	)
	if err != nil {
		return "", fmt.Errorf("CreateFile for volume '%s' failed: %w (ensure admin rights if needed)", volumeDevicePath, err)
	}
	defer windows.CloseHandle(hVolume)

	extentsBufferSize := uint32(unsafe.Sizeof(VolumeDiskExtents{}) + 3*unsafe.Sizeof(DiskExtent{}))
	extentsBuffer := make([]byte, extentsBufferSize)
	var bytesReturned uint32

	err = windows.DeviceIoControl(
		hVolume,
		IOCTL_VOLUME_GET_VOLUME_DISK_EXTENTS,
		nil, 0,
		&extentsBuffer[0], extentsBufferSize,
		&bytesReturned, nil,
	)
	if err != nil {
		return "", fmt.Errorf("DeviceIoControl IOCTL_VOLUME_GET_VOLUME_DISK_EXTENTS on '%s' failed: %w", volumeDevicePath, err)
	}

	if bytesReturned < uint32(unsafe.Offsetof(VolumeDiskExtents{}.Extents)) {
		return "", fmt.Errorf("IOCTL_VOLUME_GET_VOLUME_DISK_EXTENTS on '%s' returned insufficient data (%d bytes) for NumberOfDiskExtents field", volumeDevicePath, bytesReturned)
	}

	diskExtents := (*VolumeDiskExtents)(unsafe.Pointer(&extentsBuffer[0]))
	if diskExtents.NumberOfDiskExtents == 0 {
		return "", fmt.Errorf("no disk extents found for volume '%s'", volumeDevicePath)
	}

	minRequiredSize := uint32(unsafe.Offsetof(VolumeDiskExtents{}.Extents)) + (diskExtents.NumberOfDiskExtents * uint32(unsafe.Sizeof(DiskExtent{})))
	if bytesReturned < minRequiredSize {
		return "", fmt.Errorf("IOCTL_VOLUME_GET_VOLUME_DISK_EXTENTS on '%s' returned insufficient data (%d bytes) for %d extents", volumeDevicePath, bytesReturned, diskExtents.NumberOfDiskExtents)
	}

	diskNumber := diskExtents.Extents[0].DiskNumber
	physicalDrivePath := fmt.Sprintf("\\\\.\\PhysicalDrive%d", diskNumber)

	return getDiskSerialNumberForPath(physicalDrivePath)
}

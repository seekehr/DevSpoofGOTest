package native

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

func GetSystemUUID() (string, error) {
	// First call with 0 buffer size to get required size
	size, _, _ := procGetSystemFirmwareTable.Call(
		uintptr(RSMB),
		0,
		0,
		0,
	)

	if size == 0 {
		return "", fmt.Errorf("failed to get required buffer size for SMBIOS data")
	}

	buffer := make([]byte, size)

	// Second call to get the actual data
	result, _, _ := procGetSystemFirmwareTable.Call(
		uintptr(RSMB),
		0,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(size),
	)

	if result == 0 {
		return "", fmt.Errorf("failed to get firmware table data")
	}

	smbiosData := (*RawSMBIOSData)(unsafe.Pointer(&buffer[0]))
	tableData := buffer[unsafe.Offsetof(smbiosData.SMBIOSTableData):]
	tableData = tableData[:smbiosData.Length]

	return parseSystemUUID(tableData)
}

func parseSystemUUID(data []byte) (string, error) {
	offset := 0
	for offset < len(data) {
		if offset+4 > len(data) {
			break
		}

		header := (*SMBIOSHeader)(unsafe.Pointer(&data[offset]))

		if header.Type == SmbiosTypeSystemInformation {
			// Minimum length for Type 1 structure with UUID is 0x19 (25 bytes)
			// header (4 bytes) + fixed fields (21 bytes including UUID)
			const minType1LengthWithUUID = 0x19
			if int(header.Length) < minType1LengthWithUUID {
				return "", fmt.Errorf("system information structure (Type 1) too short (%d bytes) to contain UUID (requires at least %d)", header.Length, minType1LengthWithUUID)
			}

			// The UUID is at offset 0x08 from the start of the formatted area
			uuidOffset := offset + 0x08
			if uuidOffset+16 > len(data) { // UUID is 16 bytes
				return "", fmt.Errorf("invalid UUID data length in System Information structure")
			}

			uuidBytes := data[uuidOffset : uuidOffset+16]

			// Format the UUID bytes into the standard string representation
			return fmt.Sprintf("%02X%02X%02X%02X-%02X%02X-%02X%02X-%02X%02X-%02X%02X%02X%02X%02X%02X",
				uuidBytes[3], uuidBytes[2], uuidBytes[1], uuidBytes[0],
				uuidBytes[5], uuidBytes[4],
				uuidBytes[7], uuidBytes[6],
				uuidBytes[8], uuidBytes[9],
				uuidBytes[10], uuidBytes[11], uuidBytes[12], uuidBytes[13], uuidBytes[14], uuidBytes[15]), nil
		}

		// Move to the next structure
		stringOffset := offset + int(header.Length)
		for stringOffset < len(data)-1 {
			if data[stringOffset] == 0 && data[stringOffset+1] == 0 {
				stringOffset += 2
				break
			}
			stringOffset++
		}

		if stringOffset >= len(data)-1 && !(len(data) >= 2 && data[len(data)-2] == 0 && data[len(data)-1] == 0) {
			break
		}

		offset = stringOffset

		if offset >= len(data) || (offset > 0 && data[offset-1] == 0 && data[offset-2] == 0 && data[offset] == 0) {
			break
		}
	}

	return "", fmt.Errorf("system information structure (Type 1) not found")
}

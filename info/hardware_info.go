package info

import (
	"bytes"
	"fmt"
	"syscall"
	"unsafe"
)

// Windows API constants
const (
	RSMB                        = 0x52534D42
	SmbiosTypeSystemInformation = 1
	SmbiosTypeBaseboard         = 2 // Type 2 is for baseboard (motherboard) info
)

type RawSMBIOSData struct {
	Used20CallingMethod byte
	MajorVersion        byte
	MinorVersion        byte
	DmiRevision         byte
	Length              uint32
	SMBIOSTableData     [1]byte // This is actually a variable length array
}

type SMBIOSHeader struct {
	Type   byte
	Length byte
	Handle uint16
}

var (
	kernel32                   = syscall.NewLazyDLL("kernel32.dll")
	procGetSystemFirmwareTable = kernel32.NewProc("GetSystemFirmwareTable")
)

func GetMotherboardSerial() (string, error) {
	// First call with 0 buffer size to get required size
	size, _, _ := procGetSystemFirmwareTable.Call(
		uintptr(RSMB),
		0,
		0,
		0,
	)

	if size == 0 {
		return "", fmt.Errorf("failed to get required buffer size")
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

	// Parse the SMBIOS data to find type 2 (baseboard) structure
	smbiosData := (*RawSMBIOSData)(unsafe.Pointer(&buffer[0]))
	// The actual SMBIOS table data starts at SMBIOSTableData
	tableData := buffer[unsafe.Offsetof(smbiosData.SMBIOSTableData):]
	// Limit to actual data length reported in the header
	tableData = tableData[:smbiosData.Length]

	return parseBaseboardSerial(tableData)
}

func parseBaseboardSerial(data []byte) (string, error) {
	offset := 0
	for offset < len(data) {
		// Ensure there's enough data for the header
		if offset+4 > len(data) {
			break
		}

		header := (*SMBIOSHeader)(unsafe.Pointer(&data[offset]))
		// Check if this is a baseboard (type 2) structure
		if header.Type == SmbiosTypeBaseboard {
			// The serial number field is typically at offset 0x07 in the fixed part
			// of the baseboard structure (after the header)
			if offset+int(header.Length) > len(data) {
				return "", fmt.Errorf("invalid baseboard structure length")
			}

			// Get the serial number string index (assumes it's at offset 0x07)
			serialIndex := data[offset+0x07] // header (4 bytes) + offset to serial

			if serialIndex == 0 {
				return "", fmt.Errorf("no serial number available")
			}

			// Find the string section (starts after the fixed structure)
			stringOffset := offset + int(header.Length)

			// Extract the Nth string (serialIndex)
			return extractString(data, stringOffset, int(serialIndex))
		}

		stringOffset := offset + int(header.Length)
		for stringOffset < len(data)-1 {
			// Check for double null terminator
			if data[stringOffset] == 0 && data[stringOffset+1] == 0 {
				stringOffset += 2
				break
			}
			stringOffset++
		}

		offset = stringOffset

		// If we're at the end of the table or hit invalid data, break
		if offset >= len(data) || data[offset] == 0 {
			break
		}
	}

	return "", fmt.Errorf("baseboard information not found")
}

func GetBIOSSerial() (string, error) { // Renamed function for clarity
	// First call with 0 buffer size to get required size
	size, _, _ := procGetSystemFirmwareTable.Call(
		uintptr(RSMB),
		0,
		0,
		0,
	)

	if size == 0 {
		return "", fmt.Errorf("failed to get required buffer size")
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

	// Parse the SMBIOS data to find type 1 (system information) structure
	smbiosData := (*RawSMBIOSData)(unsafe.Pointer(&buffer[0]))
	// The actual SMBIOS table data starts at SMBIOSTableData
	tableData := buffer[unsafe.Offsetof(smbiosData.SMBIOSTableData):]
	// Limit to actual data length reported in the header
	tableData = tableData[:smbiosData.Length]

	return parseBIOSSerial(tableData) // Call the modified parsing function
}

// Renamed and modified parsing function to target System Information (Type 1)
func parseBIOSSerial(data []byte) (string, error) {
	offset := 0
	for offset < len(data) {
		// Ensure there's enough data for the header
		if offset+4 > len(data) {
			break
		}

		header := (*SMBIOSHeader)(unsafe.Pointer(&data[offset]))

		// *** CHANGE HERE: Check if this is a System Information (type 1) structure ***
		if header.Type == SmbiosTypeSystemInformation {
			// The serial number field is typically at offset 0x07 in the fixed part
			// of the System Information structure (after the header)
			// According to the SMBIOS spec, Type 1 length is at least 0x19 for versions >= 2.4
			// and the Serial Number is at offset 0x07 from the start of the formatted area.
			if offset+int(header.Length) > len(data) {
				return "", fmt.Errorf("invalid system information structure length")
			}

			// Get the serial number string index (assumes it's at offset 0x07 relative to structure start)
			// Offset 0x07 from the start of the structure is correct for the Serial Number field in Type 1.
			serialIndex := data[offset+0x07]

			if serialIndex == 0 {
				return "", fmt.Errorf("no serial number available in System Information structure")
			}

			// Find the string section (starts after the fixed structure)
			stringOffset := offset + int(header.Length)

			// Extract the Nth string (serialIndex)
			return extractString(data, stringOffset, int(serialIndex))
		}

		// Move to the next structure
		// The end of a structure (including its strings) is marked by a double null terminator (0x00 0x00)
		stringOffset := offset + int(header.Length)
		for stringOffset < len(data)-1 {
			// Check for double null terminator
			if data[stringOffset] == 0 && data[stringOffset+1] == 0 {
				stringOffset += 2
				break
			}
			stringOffset++
		}

		// If we didn't find a double null terminator before the end of the data,
		// assume the table is truncated or malformed and break.
		if stringOffset >= len(data)-1 && !(data[len(data)-2] == 0 && data[len(data)-1] == 0) {
			break
		}

		offset = stringOffset

		// If we've advanced past the end of the data or landed on a null byte
		// right after a double null terminator (end of table), break.
		if offset >= len(data) || (offset > 0 && data[offset-1] == 0 && data[offset-2] == 0 && data[offset] == 0) {
			break
		}
	}

	return "", fmt.Errorf("system information structure (Type 1) not found")
}

func extractString(data []byte, startOffset, index int) (string, error) {
	offset := startOffset
	for i := 1; i < index; i++ {
		// Find the null terminator of the current string
		for offset < len(data) && data[offset] != 0 {
			offset++
		}
		// Skip past the null terminator
		offset++

		if offset >= len(data) {
			return "", fmt.Errorf("string index out of bounds")
		}
	}

	end := bytes.IndexByte(data[offset:], 0)
	if end == -1 {
		return "", fmt.Errorf("malformed string data")
	}

	return string(data[offset : offset+end]), nil
}

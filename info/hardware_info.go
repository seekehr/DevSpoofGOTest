package info

import (
	"bytes"
	"fmt"
	"syscall"
	"unsafe"
)

// Windows API constants
const (
	RSMB                = 0x52534D42
	SmbiosTypeBaseboard = 2 // Type 2 is for baseboard (motherboard) info
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

// GetSystemFirmwareTable Windows API function
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

	// Allocate buffer of required size
	buffer := make([]byte, size)

	// Second call to get actual data
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
	// Walk through the SMBIOS structures
	offset := 0
	for offset < len(data) {
		// Ensure there's enough data for the header
		if offset+4 > len(data) {
			break
		}

		// Get the structure header
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

		// Skip to the next structure:
		// First find the end of the current structure's string-set (terminated by double null)
		stringOffset := offset + int(header.Length)
		for stringOffset < len(data)-1 {
			// Check for double null terminator
			if data[stringOffset] == 0 && data[stringOffset+1] == 0 {
				// Move past the double null
				stringOffset += 2
				break
			}
			stringOffset++
		}

		// Move to the next structure
		offset = stringOffset

		// If we're at the end of the table or hit invalid data, break
		if offset >= len(data) || data[offset] == 0 {
			break
		}
	}

	return "", fmt.Errorf("baseboard information not found")
}

func extractString(data []byte, startOffset, index int) (string, error) {
	// Skip to the desired string
	offset := startOffset
	for i := 1; i < index; i++ {
		// Find the null terminator of the current string
		for offset < len(data) && data[offset] != 0 {
			offset++
		}
		// Skip past the null terminator
		offset++

		// Check if we've reached the end
		if offset >= len(data) {
			return "", fmt.Errorf("string index out of bounds")
		}
	}

	// Extract the string until null terminator
	end := bytes.IndexByte(data[offset:], 0)
	if end == -1 {
		return "", fmt.Errorf("malformed string data")
	}

	return string(data[offset : offset+end]), nil
}

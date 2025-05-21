package native

import (
	"bytes"
	"fmt"
	"github.com/seekehr/DevSpoofGO/logger"
	"golang.org/x/sys/windows/registry"
	"syscall"
	"unsafe"
)

// Windows API constants
const (
	RSMB                           = 0x52534D42
	SmbiosTypeSystemInformation    = 1
	SmbiosTypeBaseboard            = 2 // Type 2 is for baseboard (motherboard) native
	SmbiosTypeProcessorInformation = 4 // Type 0 is for BIOS native
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

type SmbiosProcessorInformation struct {
	SMBIOSHeader
	SocketDesignation byte // String index
	ProcessorType     byte
	Family            byte
	Manufacturer      byte    // String index
	ProcessorID       [8]byte // Interpretation is architecture-dependent
	// ... other fields follow
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

func GetBIOSSerial() (string, error) {
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

func parseBIOSSerial(data []byte) (string, error) {
	offset := 0
	for offset < len(data) {
		// Ensure there's enough data for the header
		if offset+4 > len(data) {
			break
		}

		header := (*SMBIOSHeader)(unsafe.Pointer(&data[offset]))
		// type 1 ;3
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

		if stringOffset >= len(data)-1 && !(data[len(data)-2] == 0 && data[len(data)-1] == 0) {
			break
		}

		offset = stringOffset

		if offset >= len(data) || (offset > 0 && data[offset-1] == 0 && data[offset-2] == 0 && data[offset] == 0) {
			break
		}
	}

	return "", fmt.Errorf("system information structure (Type 1) not found")
}

func GetProcessorID() (string, error) {
	// First call with 0 buffer size to get required size
	size, _, err := procGetSystemFirmwareTable.Call(
		uintptr(RSMB),
		0,
		0,
		0,
	)

	if size == 0 {
		if err != nil {
			return "", fmt.Errorf("failed to get required buffer size for SMBIOS data: %w", err)
		}
		return "", fmt.Errorf("failed to get required buffer size for SMBIOS data: size is 0")
	}

	buffer := make([]byte, size)

	// Second call to get the actual data
	bytesWritten, _, err := procGetSystemFirmwareTable.Call(
		uintptr(RSMB),
		0,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(size),
	)

	if bytesWritten == 0 {
		if err != nil {
			return "", fmt.Errorf("failed to get firmware table data: %w", err)
		}
		return "", fmt.Errorf("failed to get firmware table data: 0 bytes written")
	}

	if bytesWritten < 8 {
		return "", fmt.Errorf("received truncated SMBIOS data (less than header size)")
	}

	smbiosData := (*RawSMBIOSData)(unsafe.Pointer(&buffer[0]))

	tableDataOffset := unsafe.Offsetof(smbiosData.SMBIOSTableData)
	if int(tableDataOffset) > len(buffer) {
		return "", fmt.Errorf("internal error: invalid SMBIOS data buffer offset")
	}
	tableData := buffer[tableDataOffset:]

	if int(smbiosData.Length) > len(tableData) {
		return "", fmt.Errorf("SMBIOS reported length (%d) exceeds buffer capacity (%d)", smbiosData.Length, len(tableData))
	}
	tableData = tableData[:smbiosData.Length]

	offset := 0
	for offset < len(tableData) {
		if offset+4 > len(tableData) {
			break
		}

		header := (*SMBIOSHeader)(unsafe.Pointer(&tableData[offset]))

		if header.Type == SmbiosTypeProcessorInformation {
			if offset+16 > len(tableData) || int(header.Length) < 16 {
				return "", fmt.Errorf("processor information structure (Type 4) too short (%d bytes) to contain ProcessorID (requires at least 16)", header.Length)
			}

			processorInfo := (*SmbiosProcessorInformation)(unsafe.Pointer(&tableData[offset]))

			// Format ProcessorID bytes as a hexadecimal string
			return fmt.Sprintf("%x", processorInfo.ProcessorID[:]), nil
		}

		structSize := int(header.Length)
		stringOffset := offset + structSize

		for stringOffset+1 < len(tableData) {
			if tableData[stringOffset] == 0 && tableData[stringOffset+1] == 0 {
				structSize += (stringOffset - (offset + int(header.Length))) + 2
				break
			}
			stringOffset++
		}

		if stringOffset+1 >= len(tableData) {
			if len(tableData) >= 2 && tableData[len(tableData)-2] == 0 && tableData[len(tableData)-1] == 0 {
				structSize += len(tableData) - (offset + int(header.Length))
			} else {
				structSize = int(header.Length)
			}
		}

		offset += structSize

		if offset >= len(tableData) {
			break
		}
	}

	return "", fmt.Errorf("processor information structure (Type 4) not found")
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

func GetMachineGUID() (string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.READ)
	if err != nil {
		return "", err
	}
	defer k.Close()

	s, _, err := k.GetStringValue("MachineGuid")
	if err != nil {
		logger.Error("Failed to read MachineGuid value", err)
		return "", err
	}

	return s, nil
}

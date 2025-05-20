package info

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"golang.org/x/sys/windows/registry"
)

// getStringValue is a helper to get a string registry value.
func getStringValue(k registry.Key, name string) (string, error) {
	s, _, err := k.GetStringValue(name)
	if err != nil {
		return "", fmt.Errorf("failed to get string value '%s': %w", name, err)
	}
	return s, nil
}

// getBinaryValue is a helper to get a binary registry value, returning its hex representation.
func getBinaryValue(k registry.Key, name string) (string, error) {
	b, _, err := k.GetBinaryValue(name)
	if err != nil {
		return "", fmt.Errorf("failed to get binary value '%s': %w", name, err)
	}
	return hex.EncodeToString(b), nil
}

// HexFiletimeToFormattedTime converts a hexadecimal FILETIME string
// (e.g., "1db4d52474289ab") into a human-readable date and time string in UTC.
// It returns the formatted string and an error if parsing fails.
func HexFiletimeToFormattedTime(hexFiletime string) (string, error) {
	// Clean the hex string if it has a "0x" prefix
	if len(hexFiletime) > 2 && hexFiletime[0:2] == "0x" {
		hexFiletime = hexFiletime[2:]
	}

	// Parse the hex string into a 64-bit unsigned integer (FILETIME)
	filetimeUint, err := strconv.ParseUint(hexFiletime, 16, 64)
	if err != nil {
		return "", fmt.Errorf("failed to parse hex string '%s' as uint64: %w", hexFiletime, err)
	}

	// Constant for the difference between FILETIME epoch (1601-01-01 UTC)
	// and Unix epoch (1970-01-01 UTC) in 100-nanosecond intervals.
	const filetimeEpochDiff = 116444736000000000

	// Convert 100-nanosecond intervals to nanoseconds since Unix epoch.
	unixNano := (filetimeUint - filetimeEpochDiff) * 100

	// Create a time.Time object from nanoseconds since Unix epoch.
	t := time.Unix(0, int64(unixNano))

	// Format the time into a readable string
	return t.Format("2006-01-02 15:04:05 UTC"), nil
}

// GetDigitalID retrieves the DigitalProductId as a hexadecimal string.
func GetDigitalID(k registry.Key) (string, error) {
	digitalProductID, err := getBinaryValue(k, "DigitalProductId")
	if err != nil {
		return "", fmt.Errorf("failed to get DigitalProductId: %w", err)
	}
	return digitalProductID, nil
}

// GetDigitalID4 retrieves the DigitalProductId4 as a hexadecimal string.
func GetDigitalID4(k registry.Key) (string, error) {
	digitalProductID4, err := getBinaryValue(k, "DigitalProductId4")
	if err != nil {
		return "", fmt.Errorf("failed to get DigitalProductId4: %w", err)
	}
	return digitalProductID4, nil
}

// GetProductID retrieves the ProductId as a string.
func GetProductID(k registry.Key) (string, error) {
	productID, err := getStringValue(k, "ProductId")
	if err != nil {
		return "", fmt.Errorf("failed to get ProductId: %w", err)
	}
	return productID, nil
}

// GetInstallDate retrieves the InstallDate (Unix timestamp) and formats it as a string.
func GetInstallDate(k registry.Key) (string, error) {
	val, _, err := k.GetIntegerValue("InstallDate")
	if err != nil {
		return "", fmt.Errorf("failed to get InstallDate: %w", err)
	}

	t := time.Unix(int64(val), 0).UTC()
	return t.Format(time.RFC1123Z), nil
}

// GetInstallTime "InstallTime" is stored as a REG_QWORD (64-bit integer)
func GetInstallTime(k registry.Key) (string, error) {
	installTimeUint, _, err := k.GetIntegerValue("InstallTime")
	if err != nil {
		return "", fmt.Errorf("failed to retrieve 'InstallTime' as QWORD: %w", err)
	}

	formattedTime, err := filetimeUintToFormattedTime(installTimeUint)
	if err != nil {
		return "", fmt.Errorf("failed to decode 'InstallTime' FILETIME: %w", err)
	}

	return formattedTime, nil
}

func filetimeUintToFormattedTime(filetimeUint uint64) (string, error) {
	const filetimeEpochDiff = 116444736000000000

	// Convert 100-nanosecond intervals (FILETIME) to nanoseconds since Unix epoch.
	unixNano := (filetimeUint - filetimeEpochDiff) * 100

	t := time.Unix(0, int64(unixNano))

	return t.Format("2006-01-02 15:04:05 UTC"), nil
}

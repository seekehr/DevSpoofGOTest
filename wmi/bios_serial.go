package wmi

import (
	"fmt"
	"github.com/yusufpapurcu/wmi"
)

// Win32_BIOS represents the WMI class Win32_BIOS with only the fields we care about.
type Win32_BIOS struct {
	SerialNumber string
}

// GetBIOSSerial fetches the BIOS serial number using WMI.
func GetBIOSSerial() (string, error) {
	fmt.Println("hi")
	var biosInfo []Win32_BIOS
	query := wmi.CreateQuery(&biosInfo, "")
	fmt.Printf("\nAttempting to query WMI: %s\n", query)
	err := wmi.Query(query, &biosInfo)
	fmt.Println("no")
	if err != nil {
		return "", fmt.Errorf("WMI query failed: %w", err)
	}

	if len(biosInfo) == 0 {
		return "", fmt.Errorf("no BIOS information found")
	}

	return biosInfo[0].SerialNumber, nil
}

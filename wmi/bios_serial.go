package wmi

import (
	"fmt"
	"github.com/yusufpapurcu/wmi"
)

// Win32_BIOS represents the WMI class Win32_BIOS with only the fields we care about.
type Win32_BIOS struct {
	SerialNumber string
	Name         string
}

// GetBIOSSerial fetches the BIOS serial number using WMI.
func GetBIOSSerial() (Win32_BIOS, error) {
	var bios []Win32_BIOS
	err := wmi.Query("SELECT SerialNumber FROM Win32_BIOS", &bios)
	if err != nil {
		return Win32_BIOS{}, fmt.Errorf("WMI query failed: %w", err)
	}
	if len(bios) > 0 {
		return bios[0], nil
	}
	return Win32_BIOS{}, fmt.Errorf("no BIOS information found")
}

func GetBIOSSerialFromAll() (string, error) {
	var biosInfo []Win32_BIOS
	query := wmi.CreateQuery(&biosInfo, "")
	err := wmi.Query(query, &biosInfo)
	if err != nil {
		return "", fmt.Errorf("WMI query failed: %w", err)
	}

	if len(biosInfo) == 0 {
		return "", fmt.Errorf("no BIOS information found")
	}

	return biosInfo[0].SerialNumber, nil
}

package wmi

import (
	"fmt"
	"github.com/yusufpapurcu/wmi"
)

type Win32_PhysicalMemory struct {
	SerialNumber string
	PartNumber   string
}

func GetPhysicalMemoryInfo() (Win32_PhysicalMemory, error) {
	var physicalMemory []Win32_PhysicalMemory
	err := wmi.Query("SELECT SerialNumber, PartNumber FROM Win32_PhysicalMemory", &physicalMemory)
	if err != nil {
		return Win32_PhysicalMemory{}, fmt.Errorf("WMI query failed: %w", err)
	}
	if len(physicalMemory) > 0 {
		return physicalMemory[0], nil
	}
	return Win32_PhysicalMemory{}, fmt.Errorf("no physical memory information found")
}

func GetPhysicalMemoryInfoFromAll() (Win32_PhysicalMemory, error) {
	var physicalMemory []Win32_PhysicalMemory
	query := wmi.CreateQuery(&physicalMemory, "")
	err := wmi.Query(query, &physicalMemory)
	if err != nil {
		return Win32_PhysicalMemory{}, fmt.Errorf("WMI query failed: %w", err)
	}

	if len(physicalMemory) == 0 {
		return Win32_PhysicalMemory{}, fmt.Errorf("no physical memory information found")
	}

	return physicalMemory[0], nil
}

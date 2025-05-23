package wmi

import (
	"fmt"
	"github.com/yusufpapurcu/wmi"
)

type Win32_Processor struct {
	SerialNumber string
	ProcessorId  string
}

func GetProcessorInfo() (Win32_Processor, error) {
	var processor []Win32_Processor
	err := wmi.Query("SELECT SerialNumber, ProcessorId FROM Win32_Processor", &processor)
	if err != nil {
		return Win32_Processor{}, fmt.Errorf("WMI query failed: %w", err)
	}
	if len(processor) > 0 {
		return processor[0], nil
	}
	return Win32_Processor{}, fmt.Errorf("no processor information found")
}

func GetProcessorInfoFromAll() (Win32_Processor, error) {
	var processor []Win32_Processor
	query := wmi.CreateQuery(&processor, "")
	err := wmi.Query(query, &processor)
	if err != nil {
		return Win32_Processor{}, fmt.Errorf("WMI query failed: %w", err)
	}

	if len(processor) == 0 {
		return Win32_Processor{}, fmt.Errorf("no processor information found")
	}

	return processor[0], nil
}

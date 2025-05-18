package info

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

// bytePointerToString converts a null-terminated byte pointer to a Go string
func bytePointerToString(ptr *byte) string {
	if ptr == nil {
		return ""
	}
	
	length := 0
	for {
		if *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + uintptr(length))) == 0 {
			break
		}
		length++
	}
	
	bytes := make([]byte, length)
	for i := 0; i < length; i++ {
		bytes[i] = *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + uintptr(i)))
	}
	
	return string(bytes)
}

// WlanInfo represents network adapter information
type WlanInfo struct {
	MAC         string
	GUID        string
	BSSID       string
	AdapterType string
	Description string
}

// GetWlanInfo retrieves network adapter information
func GetWlanInfo() ([]WlanInfo, error) {
	adapters, err := getAdaptersFromIpHelper()
	if err != nil {
		return nil, err
	}

	tryAddWlanInfo(adapters)
	return filterAdapters(adapters), nil
}

// normalizeGUID removes formatting characters from GUIDs
func normalizeGUID(guid string) string {
	guid = strings.ToLower(guid)
	guid = strings.ReplaceAll(guid, "{", "")
	guid = strings.ReplaceAll(guid, "}", "")
	guid = strings.ReplaceAll(guid, "-", "")
	return guid
}

// filterAdapters removes duplicate and uninteresting adapters
func filterAdapters(adapters []WlanInfo) []WlanInfo {
	seenMACs := make(map[string]bool)
	bestAdapters := make(map[string]WlanInfo)
	
	for _, adapter := range adapters {
		if adapter.MAC == "" || adapter.MAC == "00:00:00:00:00:00" {
			continue
		}
		
		if adapter.AdapterType == "Wi-Fi" && adapter.BSSID != "" {
			bestAdapters[adapter.MAC] = adapter
			seenMACs[adapter.MAC] = true
			continue
		}
		
		if !seenMACs[adapter.MAC] {
			bestAdapters[adapter.MAC] = adapter
			seenMACs[adapter.MAC] = true
		} else {
			existing := bestAdapters[adapter.MAC]
			if adapter.AdapterType == "Wi-Fi" && existing.AdapterType != "Wi-Fi" {
				bestAdapters[adapter.MAC] = adapter
			}
		}
	}
	
	var result []WlanInfo
	for _, adapter := range bestAdapters {
		result = append(result, adapter)
	}
	
	return result
}

// getAdaptersFromIpHelper gets adapter information using IP Helper API
func getAdaptersFromIpHelper() ([]WlanInfo, error) {
	var adapters []WlanInfo
	
	iphlpapi := syscall.NewLazyDLL("iphlpapi.dll")
	if iphlpapi.Load() != nil {
		return nil, fmt.Errorf("failed to load iphlpapi.dll")
	}

	getAdaptersAddresses := iphlpapi.NewProc("GetAdaptersAddresses")
	
	const (
		MAX_ADAPTER_ADDRESS_LENGTH = 8
		ERROR_BUFFER_OVERFLOW      = 111
		GAA_FLAG_INCLUDE_GATEWAYS  = 0x1
		GAA_FLAG_INCLUDE_ALL_INTERFACES = 0x100
		AF_UNSPEC                  = 0
		
		IF_TYPE_ETHERNET_CSMACD    = 6
		IF_TYPE_ISO88025_TOKENRING = 9
		IF_TYPE_PPP                = 23
		IF_TYPE_SOFTWARE_LOOPBACK  = 24
		IF_TYPE_IEEE80211          = 71
		IF_TYPE_TUNNEL             = 131
		IF_TYPE_IEEE1394           = 144
	)

	var size uint32
	result, _, _ := getAdaptersAddresses.Call(
		uintptr(AF_UNSPEC),
		uintptr(GAA_FLAG_INCLUDE_GATEWAYS | GAA_FLAG_INCLUDE_ALL_INTERFACES),
		0, 0,
		uintptr(unsafe.Pointer(&size)),
	)

	if result != uintptr(ERROR_BUFFER_OVERFLOW) {
		return nil, fmt.Errorf("GetAdaptersAddresses failed with %d", result)
	}

	buffer := make([]byte, size)
	result, _, _ = getAdaptersAddresses.Call(
		uintptr(AF_UNSPEC),
		uintptr(GAA_FLAG_INCLUDE_GATEWAYS | GAA_FLAG_INCLUDE_ALL_INTERFACES),
		0,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(&size)),
	)

	if result != 0 {
		return nil, fmt.Errorf("GetAdaptersAddresses failed with %d", result)
	}

	type IP_ADAPTER_ADDRESSES struct {
		Length                uint32
		IfIndex               uint32
		Next                  uintptr
		AdapterName          *byte
		FirstUnicastAddress  uintptr
		FirstAnycastAddress  uintptr
		FirstMulticastAddress uintptr
		FirstDnsServerAddress uintptr
		DnsSuffix            *uint16
		Description          *uint16
		FriendlyName         *uint16
		PhysicalAddress      [MAX_ADAPTER_ADDRESS_LENGTH]byte
		PhysicalAddressLength uint32
		Flags                 uint32
		Mtu                   uint32
		IfType                uint32
		OperStatus            uint32
		Ipv6IfIndex           uint32
		ZoneIndices           [16]uint32
		FirstPrefix           uintptr
	}

	for adapterPtr := uintptr(unsafe.Pointer(&buffer[0])); adapterPtr != 0; {
		adapter := (*IP_ADAPTER_ADDRESSES)(unsafe.Pointer(adapterPtr))
		
		mac := ""
		if adapter.PhysicalAddressLength > 0 {
			for i := 0; i < int(adapter.PhysicalAddressLength); i++ {
				if i > 0 {
					mac += ":"
				}
				mac += fmt.Sprintf("%02X", adapter.PhysicalAddress[i])
			}
		}

		guid := ""
		if adapter.AdapterName != nil {
			guid = bytePointerToString(adapter.AdapterName)
		}
		
		adapterType := "Unknown"
		switch adapter.IfType {
		case IF_TYPE_ETHERNET_CSMACD:
			adapterType = "Ethernet"
		case IF_TYPE_IEEE80211:
			adapterType = "Wi-Fi"
		case IF_TYPE_SOFTWARE_LOOPBACK:
			adapterType = "Loopback"
		case IF_TYPE_TUNNEL:
			adapterType = "Tunnel"
		case IF_TYPE_PPP:
			adapterType = "PPP"
		}
		
		if strings.HasPrefix(mac, "00:15:5D") {
			adapterType = "Virtual"
		}

		description := ""
		if adapter.Description != nil {
			var descriptionBytes []uint16
			for i := 0; ; i += 2 {
				char := *(*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(adapter.Description)) + uintptr(i)))
				if char == 0 {
					break
				}
				descriptionBytes = append(descriptionBytes, char)
			}
			description = syscall.UTF16ToString(descriptionBytes)
		}

		if adapter.PhysicalAddressLength > 0 {
			adapters = append(adapters, WlanInfo{
				MAC:         mac,
				GUID:        guid,
				BSSID:       "",
				AdapterType: adapterType,
				Description: description,
			})
		}
		
		adapterPtr = adapter.Next
	}
	
	return adapters, nil
}

// tryAddWlanInfo adds WLAN-specific information to existing adapters
func tryAddWlanInfo(adapters []WlanInfo) {
	wlanapi := syscall.NewLazyDLL("wlanapi.dll")
	if wlanapi.Load() != nil {
		return
	}
	
	wlanOpenHandle := wlanapi.NewProc("WlanOpenHandle")
	wlanEnumInterfaces := wlanapi.NewProc("WlanEnumInterfaces")
	wlanQueryInterface := wlanapi.NewProc("WlanQueryInterface")
	wlanFreeMemory := wlanapi.NewProc("WlanFreeMemory")
	wlanCloseHandle := wlanapi.NewProc("WlanCloseHandle")

	var clientVersion uint32 = 2
	var clientHandle uintptr
	var negotiatedVersion uint32
	
	result, _, _ := wlanOpenHandle.Call(
		uintptr(clientVersion),
		0,
		uintptr(unsafe.Pointer(&negotiatedVersion)),
		uintptr(unsafe.Pointer(&clientHandle)),
	)
	
	if result != 0 {
		return
	}
	
	defer wlanCloseHandle.Call(clientHandle, 0)
	
	var interfaceList uintptr
	result, _, _ = wlanEnumInterfaces.Call(
		clientHandle,
		0,
		uintptr(unsafe.Pointer(&interfaceList)),
	)
	
	if result != 0 || interfaceList == 0 {
		return
	}
	
	defer wlanFreeMemory.Call(interfaceList)

	type GUID struct {
		Data1 uint32
		Data2 uint16
		Data3 uint16
		Data4 [8]byte
	}

	type WLAN_INTERFACE_INFO struct {
		InterfaceGuid           GUID
		InterfaceName           [256]uint16
		InterfaceState          uint32
	}

	type WLAN_INTERFACE_INFO_LIST struct {
		NumberOfItems uint32
		Index         uint32
		InterfaceInfo [1]WLAN_INTERFACE_INFO
	}

	const wlan_intf_opcode_current_connection = 7
	
	infoList := (*WLAN_INTERFACE_INFO_LIST)(unsafe.Pointer(interfaceList))
	numInterfaces := infoList.NumberOfItems
	
	for i := uint32(0); i < numInterfaces; i++ {
		offset := unsafe.Sizeof(WLAN_INTERFACE_INFO{}) * uintptr(i)
		infoPtr := interfaceList + unsafe.Sizeof(uint32(0))*2 + offset
		info := (*WLAN_INTERFACE_INFO)(unsafe.Pointer(infoPtr))

		guidString := fmt.Sprintf("{%08X-%04X-%04X-%02X%02X-%02X%02X%02X%02X%02X%02X}",
			info.InterfaceGuid.Data1,
			info.InterfaceGuid.Data2,
			info.InterfaceGuid.Data3,
			info.InterfaceGuid.Data4[0], info.InterfaceGuid.Data4[1],
			info.InterfaceGuid.Data4[2], info.InterfaceGuid.Data4[3],
			info.InterfaceGuid.Data4[4], info.InterfaceGuid.Data4[5],
			info.InterfaceGuid.Data4[6], info.InterfaceGuid.Data4[7])

		var dataSize uint32
		var currentConnection uintptr
		result, _, _ = wlanQueryInterface.Call(
			clientHandle,
			uintptr(unsafe.Pointer(&info.InterfaceGuid)),
			wlan_intf_opcode_current_connection,
			0,
			uintptr(unsafe.Pointer(&dataSize)),
			uintptr(unsafe.Pointer(&currentConnection)),
			0,
		)

		bssid := ""
		if result == 0 && currentConnection != 0 {
			type WLAN_ASSOCIATION_ATTRIBUTES struct {
				Dot11Ssid struct {
					Length uint32
					Ssid   [32]byte
				}
				Dot11BssType           uint32
				Dot11Bssid             [6]byte
				Dot11PhyType           uint32
				Dot11PhyIndex          uint32
				WlanSignalQuality      uint32
				RxRate                 uint32
				TxRate                 uint32
			}
			
			type WLAN_CONNECTION_ATTRIBUTES struct {
				IsState                uint32
				WlanConnectionMode     uint32
				ProfileName            [256]uint16
				AssociationAttributes  WLAN_ASSOCIATION_ATTRIBUTES
				SecurityAttributes     uint64
				MSMState               uint32
				WLAN_REASON_CODE       uint32
			}
			
			wlanConnAttr := (*WLAN_CONNECTION_ATTRIBUTES)(unsafe.Pointer(currentConnection))
			bssidBytes := wlanConnAttr.AssociationAttributes.Dot11Bssid
			
			allZeros := true
			for _, b := range bssidBytes {
				if b != 0 {
					allZeros = false
					break
				}
			}
			
			if !allZeros {
				bssid = fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X", 
					bssidBytes[0], bssidBytes[1], bssidBytes[2],
					bssidBytes[3], bssidBytes[4], bssidBytes[5])
			}
				
			wlanFreeMemory.Call(currentConnection)
		}

		if bssid != "" {
			normalizedGUID := normalizeGUID(guidString)
			
			for j := range adapters {
				if normalizeGUID(adapters[j].GUID) == normalizedGUID {
					adapters[j].BSSID = bssid
					adapters[j].AdapterType = "Wi-Fi"
					break
				}
			}
		}
	}
}
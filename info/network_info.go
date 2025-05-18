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
	
	// Find string length by searching for null terminator
	length := 0
	for {
		if *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + uintptr(length))) == 0 {
			break
		}
		length++
	}
	
	// Convert byte slice to string
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
	// Get all network adapters
	adapters, err := getAdaptersFromIpHelper()
	if err != nil {
		return nil, err
	}

	// Try to add WLAN-specific information where available
	// This modifies the existing adapters array
	tryAddWlanInfo(adapters)
	
	// Filter adapters to reduce redundancy and remove virtual interfaces
	filteredAdapters := filterAdapters(adapters)
	
	return filteredAdapters, nil
}

// normalizeGUID removes formatting characters from GUIDs to enable better comparison
func normalizeGUID(guid string) string {
	// Remove braces, hyphens and convert to lowercase
	guid = strings.ToLower(guid)
	guid = strings.ReplaceAll(guid, "{", "")
	guid = strings.ReplaceAll(guid, "}", "")
	guid = strings.ReplaceAll(guid, "-", "")
	return guid
}

// filterAdapters removes duplicate and uninteresting adapters
func filterAdapters(adapters []WlanInfo) []WlanInfo {
	// Maps to track unique MAC addresses we've seen
	seenMACs := make(map[string]bool)
	
	// Maps to track which adapter is best for each unique MAC
	bestAdapters := make(map[string]WlanInfo)
	
	// First pass - identify connected adapters and best candidates
	for _, adapter := range adapters {
		// Skip empty MACs
		if adapter.MAC == "" || adapter.MAC == "00:00:00:00:00:00" {
			continue
		}
		
		// Prefer connected Wi-Fi adapters
		if adapter.AdapterType == "Wi-Fi" && adapter.BSSID != "" {
			bestAdapters[adapter.MAC] = adapter
			seenMACs[adapter.MAC] = true
			continue
		}
		
		// For non-connected adapters, prefer the one with the shortest GUID (likely the primary)
		// or the first one we see
		if !seenMACs[adapter.MAC] {
			bestAdapters[adapter.MAC] = adapter
			seenMACs[adapter.MAC] = true
		} else {
			// If this adapter is Wi-Fi and the existing one isn't, prefer Wi-Fi
			existing := bestAdapters[adapter.MAC]
			if adapter.AdapterType == "Wi-Fi" && existing.AdapterType != "Wi-Fi" {
				bestAdapters[adapter.MAC] = adapter
			}
			// Otherwise, keep the existing one
		}
	}
	
	// Convert map to slice
	var result []WlanInfo
	for _, adapter := range bestAdapters {
		result = append(result, adapter)
	}
	
	return result
}

// getAdaptersFromIpHelper gets adapter information using IP Helper API
func getAdaptersFromIpHelper() ([]WlanInfo, error) {
	var adapters []WlanInfo
	
	// Load iphlpapi.dll
	iphlpapi := syscall.NewLazyDLL("iphlpapi.dll")
	if iphlpapi.Load() != nil {
		return nil, fmt.Errorf("failed to load iphlpapi.dll")
	}

	// Get GetAdaptersAddresses function
	getAdaptersAddresses := iphlpapi.NewProc("GetAdaptersAddresses")
	
	// Define constants
	const (
		MAX_ADAPTER_ADDRESS_LENGTH = 8
		MAX_ADAPTER_NAME_LENGTH    = 256
		MAX_ADAPTER_DESCRIPTION_LENGTH = 128
		ERROR_BUFFER_OVERFLOW      = 111
		GAA_FLAG_INCLUDE_GATEWAYS  = 0x1
		GAA_FLAG_INCLUDE_ALL_INTERFACES = 0x100
		AF_UNSPEC                  = 0
		
		// Interface types
		IF_TYPE_OTHER              = 1
		IF_TYPE_ETHERNET_CSMACD    = 6
		IF_TYPE_ISO88025_TOKENRING = 9
		IF_TYPE_PPP                = 23
		IF_TYPE_SOFTWARE_LOOPBACK  = 24
		IF_TYPE_IEEE80211          = 71
		IF_TYPE_TUNNEL             = 131
		IF_TYPE_IEEE1394           = 144
	)

	// First call to determine required size
	var size uint32
	result, _, _ := getAdaptersAddresses.Call(
		uintptr(AF_UNSPEC),
		uintptr(GAA_FLAG_INCLUDE_GATEWAYS | GAA_FLAG_INCLUDE_ALL_INTERFACES),
		0,
		0,
		uintptr(unsafe.Pointer(&size)),
	)

	if result != uintptr(ERROR_BUFFER_OVERFLOW) {
		return nil, fmt.Errorf("GetAdaptersAddresses failed with %d", result)
	}

	// Allocate buffer
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

	// Define IP_ADAPTER_ADDRESSES structure (simplified)
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
		// More fields exist but not needed for this implementation
	}

	// Process each adapter
	for adapterPtr := uintptr(unsafe.Pointer(&buffer[0])); adapterPtr != 0; {
		adapter := (*IP_ADAPTER_ADDRESSES)(unsafe.Pointer(adapterPtr))
		
		// Get MAC address (physical address)
		mac := ""
		if adapter.PhysicalAddressLength > 0 {
			for i := 0; i < int(adapter.PhysicalAddressLength); i++ {
				if i > 0 {
					mac += ":"
				}
				mac += fmt.Sprintf("%02X", adapter.PhysicalAddress[i])
			}
		}

		// Get adapter name (GUID)
		guid := ""
		if adapter.AdapterName != nil {
			guid = bytePointerToString(adapter.AdapterName)
		}

		// For now, leave BSSID empty for general adapters
		bssid := ""
		
		// Get adapter type
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
		
		// Determine if it's a virtual adapter based on MAC
		if strings.HasPrefix(mac, "00:15:5D") {
			adapterType = "Virtual"
		}

		// Get adapter description
		description := ""
		if adapter.Description != nil {
			// Convert UTF-16 to string
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

		// Only add adapters with physical addresses
		if adapter.PhysicalAddressLength > 0 {
			adapters = append(adapters, WlanInfo{
				MAC:         mac,
				GUID:        guid,
				BSSID:       bssid,
				AdapterType: adapterType,
				Description: description,
			})
		}
		
		// Move to next adapter
		adapterPtr = adapter.Next
	}
	
	return adapters, nil
}

// tryAddWlanInfo adds WLAN-specific information to existing adapters
func tryAddWlanInfo(adapters []WlanInfo) {
	// Load wlanapi.dll
	wlanapi := syscall.NewLazyDLL("wlanapi.dll")
	if wlanapi.Load() != nil {
		return
	}
	
	// Get WlanOpenHandle function
	wlanOpenHandle := wlanapi.NewProc("WlanOpenHandle")
	
	// Get WlanEnumInterfaces function
	wlanEnumInterfaces := wlanapi.NewProc("WlanEnumInterfaces")

	// Get WlanQueryInterface function for BSSID
	wlanQueryInterface := wlanapi.NewProc("WlanQueryInterface")
	
	// Get WlanFreeMemory function
	wlanFreeMemory := wlanapi.NewProc("WlanFreeMemory")
	
	// Get WlanCloseHandle function
	wlanCloseHandle := wlanapi.NewProc("WlanCloseHandle")

	// Open WLAN handle
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
	
	// Ensure we close the handle when done
	defer wlanCloseHandle.Call(clientHandle, 0)
	
	// Enumerate WLAN interfaces
	var interfaceList uintptr
	result, _, _ = wlanEnumInterfaces.Call(
		clientHandle,
		0,
		uintptr(unsafe.Pointer(&interfaceList)),
	)
	
	if result != 0 || interfaceList == 0 {
		return
	}
	
	// Free the memory when we're done
	defer wlanFreeMemory.Call(interfaceList)

	// Define the WLAN_INTERFACE_INFO_LIST structure
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

	// Access the interface list
	infoList := (*WLAN_INTERFACE_INFO_LIST)(unsafe.Pointer(interfaceList))
	numInterfaces := infoList.NumberOfItems

	// Define the types we need for querying BSSID
	type DOT11_MAC_ADDRESS [6]byte
	
	// Define the opcode for querying BSSID
	const wlan_intf_opcode_current_connection = 7
	const wlan_intf_opcode_bssid_list = 8
	
	// Process each interface
	for i := uint32(0); i < numInterfaces; i++ {
		// Calculate the offset to this interface info
		offset := unsafe.Sizeof(WLAN_INTERFACE_INFO{}) * uintptr(i)
		infoPtr := interfaceList + unsafe.Sizeof(uint32(0))*2 + offset
		info := (*WLAN_INTERFACE_INFO)(unsafe.Pointer(infoPtr))

		// Get interface GUID as string in format {xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx}
		guidString := fmt.Sprintf("{%08X-%04X-%04X-%02X%02X-%02X%02X%02X%02X%02X%02X}",
			info.InterfaceGuid.Data1,
			info.InterfaceGuid.Data2,
			info.InterfaceGuid.Data3,
			info.InterfaceGuid.Data4[0], info.InterfaceGuid.Data4[1],
			info.InterfaceGuid.Data4[2], info.InterfaceGuid.Data4[3],
			info.InterfaceGuid.Data4[4], info.InterfaceGuid.Data4[5],
			info.InterfaceGuid.Data4[6], info.InterfaceGuid.Data4[7])

		// Query for BSSID using current connection
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

		// Get BSSID
		bssid := ""
		if result == 0 && currentConnection != 0 {
			// Define WLAN_CONNECTION_ATTRIBUTES structure layout
			type WLAN_ASSOCIATION_ATTRIBUTES struct {
				Dot11Ssid struct {
					Length uint32
					Ssid   [32]byte
				}
				Dot11BssType           uint32
				Dot11Bssid             [6]byte // This is the BSSID
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
				SecurityAttributes     uint64 // Simplified, actually a struct
				MSMState               uint32
				WLAN_REASON_CODE       uint32
			}
			
			// Cast to WLAN_CONNECTION_ATTRIBUTES
			wlanConnAttr := (*WLAN_CONNECTION_ATTRIBUTES)(unsafe.Pointer(currentConnection))
			
			// Get BSSID directly from the structure
			bssidBytes := wlanConnAttr.AssociationAttributes.Dot11Bssid
			
			// Format BSSID as a MAC address string only if it's not all zeros
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
				
			// Free the memory for the current connection info
			wlanFreeMemory.Call(currentConnection)
		}

		// If we found a BSSID, update the corresponding adapter
		if bssid != "" {
			// Find the corresponding adapter in our list
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

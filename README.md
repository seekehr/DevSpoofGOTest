# DevSpoofGOTest

Testing project for: https://github.com/seekehr/DevSpoofGO

Run with `go run`. Then keep this running in another terminal window and run  `go run` for DevSpoofGo and enter `DevSpoofGOTest.exe` when asked for the program to inject into  

## Flags
- `-o` For OS information (e.g hostname)
- `-h` For hardware information (e.g bios serial, motherboard serial, processor id, etc)
- `-d` For disk information (e.g volume serial)
- `-n` For network information (e.g wlan GUID, mac addr, BSSID, etc)
- `-w` for WMI (e.g processor id)
- `-r` for registry (e.g certificates info)
Full command: `go run . -o -h -d -n -w -r`

**Current lines:** 1626
`(Get-ChildItem -Recurse -Filter *.go | Get-Content).Count` (to count all lines for powershell)

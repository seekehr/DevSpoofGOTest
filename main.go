package main

import (
	"flag"
	"fmt"
	"github.com/fatih/color"
	"github.com/seekehr/DevSpoofGOTest/native"
	"github.com/seekehr/DevSpoofGOTest/wmi"
	"golang.org/x/sys/windows/registry"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	getComputerNameA = kernel32.NewProc("GetComputerNameA")
	getComputerNameW = kernel32.NewProc("GetComputerNameW")
)

func main() {

	// Parse command-line flags
	osFlag := flag.Bool("o", false, "enable os output")
	diskFlag := flag.Bool("d", false, "enable disk output")
	hardwareFlag := flag.Bool("h", false, "enable hardware output")
	networkFlag := flag.Bool("n", false, "enable network output")
	certificatesFlag := flag.Bool("c", false, "enable certificate output")
	versionInfoFlag := flag.Bool("v", false, "enable version native output")
	wmiFlag := flag.Bool("w", false, "enable WMI output")
	flag.Parse()

	var activeFlags []string
	if *osFlag {
		activeFlags = append(activeFlags, "o")
	}
	if *diskFlag {
		activeFlags = append(activeFlags, "d")
	}
	if *hardwareFlag {
		activeFlags = append(activeFlags, "h")
	}
	if *networkFlag {
		activeFlags = append(activeFlags, "n")
	}
	if *certificatesFlag {
		activeFlags = append(activeFlags, "c")
	}
	if *versionInfoFlag {
		activeFlags = append(activeFlags, "v")
	}
	if *wmiFlag {
		activeFlags = append(activeFlags, "w")
	}

	i := 0
	for {
		OutputProcess(i, activeFlags...)
		time.Sleep(4 * time.Second)
		i++
	}
}

var red = color.New(color.FgRed).SprintFunc()
var green = color.New(color.FgGreen).SprintFunc()
var blue = color.New(color.FgBlue).SprintFunc()
var cyan = color.New(color.FgCyan).SprintFunc()

func OutputProcess(iteration int, flags ...string) {
	// Get the current process information
	process := os.Getpid()
	processName, err := os.Executable()
	if err != nil {
		fmt.Println("Error getting process name:", err)
		return
	}

	// Extract just the base name from the full path
	processName = filepath.Base(processName)
	fmt.Println("=========" + blue("DevSpoofGOTest.exe "+strconv.Itoa(iteration)) + "=========")
	fmt.Println(green("PID: ") + strconv.Itoa(process))

	for _, aflag := range flags {
		if aflag == "o" {
			outputOS()
		} else if aflag == "d" {
			outputDisk()
		} else if aflag == "h" {
			outputHardware()
		} else if aflag == "n" {
			outputNetwork()
		} else if aflag == "c" {
			outputCertificates()
		} else if aflag == "v" {
			outputVersionInfo()
		} else if aflag == "w" {
			outputWMI()
		} else {
			fmt.Println(red("Invalid flag: " + aflag))
		}
	}

	fmt.Println("===========================================\n")
}

func outputOS() {
	computerNameA, errCompA := native.GetComputerNameA()
	computerNameW, errCompW := native.GetComputerNameW()

	str := green("PC name: ")
	if errCompA != nil {
		str += red("Error getting computer name ("+errCompA.Error()+")") + " || "
	} else {
		str += computerNameA + cyan(" || ")
	}
	if errCompW != nil {
		str += red("Error getting computer name (" + errCompW.Error() + ")")
	} else {
		str += computerNameW
	}
	fmt.Println(str)
}

func outputDisk() {
	volumeSerialA, errVolA := native.GetVolumeSerialA()
	volumeSerialW, errVolW := native.GetVolumeSerialW()

	str := green("Volume Serial: ")
	if errVolA != nil {
		str += red("Error getting computer name ("+errVolA.Error()+")") + " || "
	} else {
		str += volumeSerialA + cyan(" || ")
	}
	if errVolW != nil {
		str += red("Error getting computer name (" + errVolW.Error() + ")")
	} else {
		str += volumeSerialW
	}

	diskSerial, err := native.GetActiveDriveSerialNumber()
	str += "\n" + green("Disk Serial: ")
	if err != nil {
		str += red("Error getting disk serial: " + err.Error())
	} else {
		str += diskSerial
	}
	fmt.Println(str)
}

func outputHardware() {
	motherboardSerial, err := native.GetMotherboardSerial()

	str := green("Motherboard Serial: ")
	if err != nil {
		str += red("Error getting motherboard serial (" + err.Error() + ")")
	} else {
		str += motherboardSerial
	}

	biosSerial, err := native.GetBIOSSerial()
	str += "\n" + green("BIOS Serial: ")
	if err != nil {
		str += red("Error getting BIOS serial (" + err.Error() + ")")
	} else {
		str += biosSerial
	}

	processorID, err := native.GetProcessorID()
	str += "\n" + green("Processor ID: ")
	if err != nil {
		str += red("Error getting processor ID (" + err.Error() + ")")
	} else {
		str += processorID
	}

	systemUUID, err := native.GetSystemUUID()
	str += "\n" + green("System UUID: ")
	if err != nil {
		str += red("Error getting system UUID (" + err.Error() + ")")
	} else {
		str += systemUUID
	}

	machineGUID, err := native.GetMachineGUID()
	str += "\n" + green("Machine GUID: ")
	if err != nil {
		str += red("Error getting machine GUID (" + err.Error() + ")")
	} else {
		str += machineGUID
	}
	fmt.Println(str)
}

func outputNetwork() {
	adapters, err := native.GetWlanInfo()

	str := green("Network Adapters: ")
	if err != nil {
		str += red("Error getting adapter native: " + err.Error())
	} else {
		if len(adapters) == 0 {
			str += cyan("No adapters found")
		} else {
			str += cyan("\n=====Network Adapters=====\n")
			adapterCount := 1
			for _, adapter := range adapters {
				// Skip virtual adapters in output
				if adapter.AdapterType == "Virtual" {
					continue
				}

				str += cyan(fmt.Sprintf("Adapter %d (%s):\n", adapterCount, adapter.AdapterType))
				str += green("MAC: ") + adapter.MAC + "\n"
				str += green("GUID: ") + adapter.GUID + "\n"

				if adapter.BSSID != "" {
					str += green("BSSID: ") + adapter.BSSID + "\n"
				} else {
					str += green("BSSID: ") + cyan("N/A") + "\n"
				}

				str += "===============\n"
				adapterCount++
			}
		}
	}
	fmt.Println(str)
}

func outputCertificates() {
	str := green("=====Certificates=====")
	certificates, err := native.GetCertificatesFromRegistry()
	if err != nil {
		str += red("Error getting certificates: " + err.Error())
	} else {
		if len(certificates) == 0 {
			str += cyan("No certificates found")
		} else {
			str += cyan("\n=====Certificates=====\n")
			for _, cert := range certificates {
				str += green("Certificate: ") + cert + "\n"
			}
			str += "=====================\n"
		}
	}

	fmt.Println(str)
}

func outputVersionInfo() {
	str := green("=====Version Info=====")
	regPath := `SOFTWARE\Microsoft\Windows NT\CurrentVersion`
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, regPath, registry.READ)
	if err != nil {
		str += red("Error getting version native: " + err.Error())
	} else {
		digitalId, err := native.GetDigitalID(k)
		if err != nil {
			str += "\n" + red("Error getting DigitalProductId: "+err.Error())
		} else {
			str += "\n" + green("DigitalProductId: ") + digitalId
		}

		digitalId4, err := native.GetDigitalID4(k)
		if err != nil {
			str += "\n" + red("Error getting DigitalProductId4: "+err.Error())
		} else {
			str += "\n" + green("DigitalProductId4: ") + digitalId4
		}

		productId, err := native.GetProductID(k)
		if err != nil {
			str += "\n" + red("Error getting ProductId: "+err.Error())
		} else {
			str += "\n" + green("ProductId: ") + productId
		}

		installDate, err := native.GetInstallDate(k)
		if err != nil {
			str += "\n" + red("Error getting InstallDate: "+err.Error())
		} else {
			str += "\n" + green("InstallDate: ") + installDate
		}

		installTime, err := native.GetInstallTime(k)
		if err != nil {
			str += "\n" + red("Error getting InstallTime: "+err.Error())
		} else {
			str += "\n" + green("InstallTime: ") + installTime
		}

		defer k.Close()
	}

	str += green("\n=====================")
	fmt.Println(str)
}

func outputWMI() {
	str := green("=====WMI=====")
	biosSerial, err := wmi.GetBIOSSerial()
	biosSerial2, err2 := wmi.GetBIOSSerialFromAll()
	if err != nil || err2 != nil {
		if err2 != nil {
			str += red("Error getting BIOS serial from all: " + err2.Error())
		} else {
			str += red("Error getting BIOS serial: " + err.Error())
		}
	} else {
		str += "\n" + green("BIOS Serial: ") + biosSerial.SerialNumber + green(" | ") + biosSerial2
		str += "\n" + green("BIOS Name: ") + biosSerial.Name
	}

	fmt.Println(str)
}

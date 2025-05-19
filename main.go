package main

import (
	"flag"
	"fmt"
	"github.com/fatih/color"
	"github.com/seekehr/DevSpoofGOTest/info"
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
		}
	}

	fmt.Println("===========================================\n")
}

func outputOS() {
	computerNameA, errCompA := info.GetComputerNameA()
	computerNameW, errCompW := info.GetComputerNameW()

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
	volumeSerialA, errVolA := info.GetVolumeSerialA()
	volumeSerialW, errVolW := info.GetVolumeSerialW()

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
	fmt.Println(str)
}

func outputHardware() {
	motherboardSerial, err := info.GetMotherboardSerial()

	str := green("Motherboard Serial: ")
	if err != nil {
		str += red("Error getting motherboard serial (" + err.Error() + ")")
	} else {
		str += motherboardSerial
	}
	fmt.Println(str)
}

func outputNetwork() {
	adapters, err := info.GetWlanInfo()

	str := green("Network Adapters: ")
	if err != nil {
		str += red("Error getting adapter info: " + err.Error())
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

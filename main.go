package main

import (
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
	i := 0
	for {
		OutputProcess(i)
		time.Sleep(4 * time.Second)
		i++
	}
}

var red = color.New(color.FgRed).SprintFunc()
var green = color.New(color.FgGreen).SprintFunc()
var blue = color.New(color.FgBlue).SprintFunc()
var cyan = color.New(color.FgCyan).SprintFunc()

func OutputProcess(iteration int) {
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

	outputOS()
	outputDisk()
	outputHardware()

	fmt.Println("=====================\n")
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

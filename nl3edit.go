// +build ignore
// run with `go run nl3edit.go <args>`

package main

import (
	"fmt"
	"github.com/malacalypse/nordlead3"
	"os"
	"strconv"
)

func printSummaryInfo(*nordlead3.PatchMemory) {
	fmt.Printf("Not yet implemented. :(")
}

func main() {
	if len(os.Args) == 1 {
		usage()
		return
	}

	filename := os.Args[1]
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Could not open %q: %q\n", filename, err)
		return
	} else {
		fmt.Printf("Opening %q\n", filename)
	}

	memory := new(nordlead3.PatchMemory)

	validFound, invalidFound, err := memory.LoadFromFile(file)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Found %v valid SysEx entries (%v invalid).\n\n", validFound, invalidFound)

	if len(os.Args) >= 3 {
		command := os.Args[2]
		depth := "3"

		switch command {
		case "print", "p":
			if len(os.Args) >= 5 {
				if len(os.Args) == 6 {
					depth = os.Args[5]
				}
				printContents(memory, os.Args[3], os.Args[4], depth)
			} else {
				fmt.Printf(memory.PrintPerformances(true))
				fmt.Printf(memory.PrintPrograms(true))
			}
		}
	}
}

func printContents(memory *nordlead3.PatchMemory, bank string, location string, depth string) {
	intBank, err := strconv.Atoi(bank)
	if err == nil {
		intLocation, err := strconv.Atoi(location)
		if err == nil {
			intDepth, err := strconv.Atoi(depth)
			if err == nil {
				performance, err := memory.GetPerformance(uint8(intBank), uint8(intLocation))
				if err == nil {
					performance.PrintContents(intDepth)
					return
				}

				program, err := memory.GetProgram(uint8(intBank), uint8(intLocation))
				if err == nil {
					program.PrintContents(intDepth)
					return
				}
			}
		}
	}
	usage()
}

func usage() {
	fmt.Println("Usage: go run nl3edit <filename.syx> [p|print] [bank] [location]")
}

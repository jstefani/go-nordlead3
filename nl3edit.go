// +build ignore
// run with `go run nl3edit.go <args>`

package main

import (
	"fmt"
	"github.com/malacalypse/nordlead3"
	"os"
)

func printSummaryInfo(*nordlead3.PatchMemory) {
	fmt.Printf("Not yet implemented. :(")
}

func main() {
	if len(os.Args) == 1 {
		fmt.Println("Usage: go run nl3edit <filename.syx>")
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
	fmt.Printf("Found %v valid SysEx entries (%v invalid).", validFound, invalidFound)

	fmt.Printf(memory.PrintPerformances(true))
	fmt.Printf(memory.PrintPrograms(true))
}

// +build ignore
// run with `go run nl3edit.go <args>`

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/malacalypse/nordlead3"
	"github.com/mitchellh/go-homedir"
)

func printSummaryInfo(*nordlead3.PatchMemory) {
	fmt.Printf("Not yet implemented. :(")
}

func printPerformance(memory *nordlead3.PatchMemory, bank int, location int, depth int) {
	performance, err := memory.GetPerformance(uint8(bank-1), uint8(location-1))
	if err != nil {
		fmt.Printf("Performance %d:%d not initialized.\n", bank, location)
		return
	}
	performance.PrintContents(depth)
}

func printProgram(memory *nordlead3.PatchMemory, bank int, location int, depth int) {
	program, err := memory.GetProgram(uint8(bank-1), uint8(location-1))
	if err != nil {
		fmt.Printf("Program %d:%d not initialized.\n", bank, location)
		return
	}
	program.PrintContents(depth)
}

func runCommands(memory *nordlead3.PatchMemory) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		// Print prompt
		fmt.Print("nl3 (h for help) > ")

		// Accept input
		scanner.Scan()
		input := scanner.Text()

		// Parse it
		args := strings.Fields(input)
		command := args[0]

		// Evaluate
		switch command {
		case "perf":
			if b, l, d, ok := getBLD(args); ok {
				printPerformance(memory, b, l, d)
			} else {
				fmt.Printf(memory.PrintPerformances(true))
			}
		case "prog":
			if b, l, d, ok := getBLD(args); ok {
				printProgram(memory, b, l, d)
			} else {
				fmt.Printf(memory.PrintPrograms(true))
			}
		case "help", "h":
			help()
		case "load", "l":
			loadFiles(memory, args[1:])
		case "quit", "q", "exit":
			fmt.Println("See ya!")
			return
		default:
			// skip
		}
	}
}

func getBLD(args []string) (bank, location, depth int, ok bool) {
	var inputs [3]string
	var outputs [3]int
	ok = true

	if len(args) == 3 {
		inputs[0] = args[1]
		inputs[1] = args[2]
		inputs[2] = "0"
		if len(args) == 4 {
			inputs[2] = args[3]
		}
	} else {
		ok = false
	}

	// make them ints
	for i, curr := range inputs {
		if val, err := strconv.Atoi(curr); err != nil {
			ok = false
		} else {
			outputs[i] = val
		}
	}

	return outputs[0], outputs[1], outputs[2], ok
}

func loadFiles(memory *nordlead3.PatchMemory, filenames []string) {
	for _, filename := range filenames {
		loadFile(memory, filename)
	}
}

func loadFile(memory *nordlead3.PatchMemory, filename string) {
	// Expand ~ character first
	expandedfn, err := homedir.Expand(filename)
	if err != nil {
		fmt.Println(err)
		return
	}
	// Detect globbing
	filenames, err := filepath.Glob(expandedfn)
	if err != nil {
		fmt.Printf("Invalid filename pattern %q.\n", filename)
		return
	}
	if len(filenames) > 1 {
		loadFiles(memory, filenames)
		return
	} else if len(filenames) == 0 {
		fmt.Printf("%q did not match any files.\n", filename)
		return
	}
	filename = filenames[0] // take the de-globbed version

	file, err := os.Open(filename)
	defer file.Close()

	if err != nil {
		fmt.Printf("Could not open %q: %q\n", filename, err)
		return
	}
	fmt.Printf("Opening %q\n", filename)

	validFound, invalidFound, err := memory.LoadFromFile(file)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Found %v valid SysEx entries (%v invalid).\n\n", validFound, invalidFound)
}

func usage() {
	fmt.Println("Usage: go run nl3edit <filename.syx>")
}

func help() {
	fmt.Println("Available commands are: ")
	fmt.Println(" h | help                                : print this help reference")
	fmt.Println(" l | load  <filename> [<filename> ...]   : load the requested file into memory")
	fmt.Println("     perf  [<bank> <location>] [<depth>] : print details of performance at that location")
	fmt.Println("     prog  [<bank> <location>] [<depth>] : print details of program at that location")
}

func main() {
	memory := new(nordlead3.PatchMemory)

	files := os.Args[1:]
	if len(files) > 0 {
		loadFiles(memory, files)
	}
	help()
	runCommands(memory)
}

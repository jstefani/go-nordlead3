// +build ignore
// run with `go run nl3edit.go <args>`

package main

import (
	"bufio"
	"errors"
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

func printPerformance(memory *nordlead3.PatchMemory, ml nordlead3.MemoryLocation, depth int) {
	performance, err := memory.GetPerformance(ml)
	if err != nil {
		fmt.Println(err)
		return
	}
	performance.PrintContents(depth)
}

func printProgram(memory *nordlead3.PatchMemory, ml nordlead3.MemoryLocation, depth int) {
	program, err := memory.GetProgram(ml)
	if err != nil {
		fmt.Println(err)
		return
	}
	program.PrintContents(depth)
}

func runCommands(memory *nordlead3.PatchMemory) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		// Print prompt
		fmt.Print("\nnl3 (h for help) > ")

		// Accept input
		scanner.Scan()
		input := scanner.Text()

		// Parse it
		args := strings.Fields(input)
		command := args[0]

		// Evaluate
		switch command {
		case "export", "e":
			if tblfn, ok := getArgs(args, []string{"string", "int", "int", "string opt"}); ok {
				export(memory, scanner, tblfn[0].(string), ml(tblfn[1].(int)-1, tblfn[2].(int)-1), tblfn[3].(string))
			} else {
				fmt.Println(" e | export  <prog|perf> <bank> <location> [<filename>]  : export bank and location to a file")
			}
		case "help", "h":
			help()
		case "load", "l":
			loadFiles(memory, args[1:])
		case "perf":
			if bld, ok := getArgs(args, []string{"int", "int", "int opt"}); ok {
				printPerformance(memory, ml(bld[0].(int)-1, bld[1].(int)-1), bld[2].(int))
			} else {
				fmt.Printf(memory.SprintPerformances(true))
			}
		case "prog":
			if bld, ok := getArgs(args, []string{"int", "int", "int opt"}); ok {
				printProgram(memory, ml(bld[0].(int)-1, bld[1].(int)-1), bld[2].(int))
			} else {
				fmt.Printf(memory.SprintPrograms(true))
			}
		case "quit", "q", "exit":
			fmt.Println("See ya!")
			return
		case "rename", "r":
			if tbln, ok := getArgs(args, []string{"string", "int", "int", "toEnd"}); ok {
				rename(memory, tbln[0].(string), ml(tbln[1].(int)-1, tbln[2].(int)-1), tbln[3].(string))
			} else {
				fmt.Println(" r | rename  <prog|perf> <bank> <location> <new name>    : rename the indicated program or performance")
			}
		default:
			if len(args) > 0 {
				fmt.Println("Invalid command. Enter h for help.")
			}
		}
	}
}

// Expectations are an array of expected types and whether or not that type is optional
// All optional arguments must go at the end and are assigned in order, no heuristics here!
// Note: A "toEnd" type is provided to capture the entire rest of the argument line as a single string
func getArgs(args []string, expectations []string) (result []interface{}, ok bool) {
	ok = true

	for i, expectation := range expectations {
		optional := false
		splexp := strings.Split(expectation, " ")
		exptype := splexp[0]
		if len(splexp) > 1 && splexp[1] == "opt" {
			optional = true
		}

		if len(args) > i+1 {
			curr := args[i+1]

			switch exptype {
			case "int":
				// make them ints
				if val, err := strconv.Atoi(curr); err != nil {
					ok = false
				} else {
					result = append(result, val)
				}
			case "string":
				result = append(result, curr)
			case "toEnd":
				result = append(result, strings.Join(args[i+1:], " "))
			default:
				// skip for now, it's unsupported
			}
		} else if optional {
			switch exptype {
			case "int":
				result = append(result, 0)
			case "string", "toEnd":
				result = append(result, "")
			default:
				// skip for now, unimplemented
			}
		} else {
			ok = false
		}
	}

	return result, ok
}

func export(memory *nordlead3.PatchMemory, scanner *bufio.Scanner, typ string, ml nordlead3.MemoryLocation, filename string) {
	var err error

	if len(filename) == 0 {
		filename, err = promptValidFilename(scanner, false)
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	// Expand ~ character first
	filename, err = homedir.Expand(filename)
	if err != nil {
		fmt.Println(err)
		return
	}

	switch typ {
	case "prog":
		err = memory.ExportProgram(ml, filename)
	case "perf":
		err = memory.ExportPerformance(ml, filename)
	}
	if err != nil {
		fmt.Printf("Error exporting %s: %s\n", typ, err)
	} else {
		fmt.Println("Done!")
	}
}

func promptValidFilename(scanner *bufio.Scanner, expectExist bool) (filename string, err error) {
outer:
	for {
		args := getPrompted("Enter filename (empty line to abort): ", scanner)
		fmt.Printf("getPrompted returned %q (len %d)\n", args, len(args))

		switch len(args) {
		case 0:
			break outer
		case 1:
			filename = args[0]
		default:
			fmt.Println("Can only accept a single filename, please try again.")
			continue
		}

		// Expand ~ character first
		filename, err := homedir.Expand(filename)
		if err != nil {
			fmt.Println(err)
			continue
		}

		// apply existing expectation
		_, err = os.Stat(filename)
		if os.IsNotExist(err) {
			if expectExist {
				fmt.Println("That file does not exist!")
				continue
			}
		} else if err != nil {
			fmt.Println("There is an error with that file: %s", err)
			continue
		} else if !expectExist {
			fmt.Println("That file already exists, cannot overwrite.")
			continue
		}

		return filename, nil
	}

	return "", errors.New("No valid filename entered. Aborting.")
}

func getPrompted(prompt string, scanner *bufio.Scanner) (args []string) {
	if scanner == nil {
		scanner = bufio.NewScanner(os.Stdin)
	}

	// Print prompt
	fmt.Print(prompt)

	// Accept input
	scanner.Scan()
	input := scanner.Text()

	// Parse it
	return strings.Fields(input)
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

	validFound, invalidFound, err := memory.ImportFrom(file)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Found %v valid SysEx entries (%v invalid).\n\n", validFound, invalidFound)
}

func rename(memory *nordlead3.PatchMemory, typ string, ml nordlead3.MemoryLocation, newName string) {
	if pt, ok := ptype(typ); ok {
		switch pt {
		case nordlead3.PerformanceT:
			p, err := memory.GetPerformance(ml)
			if err != nil {
				fmt.Printf("Performance %d:%d is not initialized.\n", ml.Bank+1, ml.Location+1)
				break
			}
			err = p.SetName(newName)
			if err != nil {
				fmt.Printf("Error renaming %d:%d (%q): %s", ml.Bank, ml.Location, p.PrintableName(), err)
				return
			}
			fmt.Println(p.Summary())
		case nordlead3.ProgramT:
			p, err := memory.GetProgram(ml)
			if err != nil {
				fmt.Printf("Program %d:%d is not initialized.\n", ml.Bank+1, ml.Location+1)
				break
			}
			err = p.SetName(newName)
			if err != nil {
				fmt.Printf("Error renaming %d:%d (%q): %s", ml.Bank, ml.Location, p.PrintableName(), err)
				return
			}
			fmt.Println(p.Summary())
		}
	}
}

func ptype(typ string) (pt nordlead3.PatchType, ok bool) {
	switch typ {
	case "prog":
		pt = nordlead3.ProgramT
	case "perf":
		pt = nordlead3.PerformanceT
	default:
		fmt.Printf("%q is not a valid type. Please use `perf` or `prog`.", typ)
		return 0, false
	}
	return pt, true
}

func ml(bank, location int) nordlead3.MemoryLocation {
	return nordlead3.MemoryLocation{bank, location}
}

func usage() {
	fmt.Println("Usage: go run nl3edit <filename.syx>")
}

func help() {
	fmt.Println("Available commands are: ")
	fmt.Println(" h | help                                                : print this help reference")
	fmt.Println(" e | export  <prog|perf> <bank> <location> [<filename>]  : export bank and location to a file")
	fmt.Println(" l | load    <filename> [<filename> ...]                 : load the requested file into memory")
	fmt.Println(" r | rename  <prog|perf> <bank> <location> <new name>    : rename the indicated program or performance")
	fmt.Println("     perf    [<bank> <location>] [<depth>]               : print details of performance at that location")
	fmt.Println("     prog    [<bank> <location>] [<depth>]               : print details of program at that location")
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

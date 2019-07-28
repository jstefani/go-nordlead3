// +build ignore
// run with `go run nl3edit.go <optional sysex filenames to preload>`

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
		case "delete", "d", "clear", "c":
			clear(memory, scanner, args[1:])
		case "export", "e":
			export(memory, scanner, args[1:])
		case "help", "h":
			help()
		case "load", "l":
			loadFiles(memory, args[1:])
		case "move", "m":
			if len(args) > 1 {
				if typ, ok := ptype(args[1]); ok {
					movePrompted(memory, scanner, typ)
				} else {
					fmt.Println(" m | move    <prog|perf>                                 : enter the move tool for programs or performances")
				}
			} else {
				fmt.Println(" m | move    <prog|perf>                                 : enter the move tool for programs or performances")
			}
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

// Command parsing helpers

// Expectations are an array of expected types and whether or not that type is optional
// All optional arguments must go at the end and are assigned in order, no heuristics here!
// A literal string match can be indicated indicating `string literal <literalval>`
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

		if len(args) > i {
			curr := args[i]

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
				result = append(result, strings.Join(args[i:], " "))
			default: // string literal
				if curr == exptype {
					// ok, don't add to received
				} else {
					ok = false
				}
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

func promptValidFilename(scanner *bufio.Scanner, expectExist bool) (filename string, err error) {
outer:
	for {
		args := getPrompted("Enter filename (empty line to abort): ", scanner)

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

// Command processing functions

func clear(memory *nordlead3.PatchMemory, scanner *bufio.Scanner, args []string) {
	if tblf, ok := getArgs(args, []string{"string", "int", "int", "string opt"}); ok {
		typ, ok := ptype(tblf[0].(string))
		ml := ml(tblf[1].(int)-1, tblf[2].(int)-1)
		force := tblf[3].(string) == "f"

		if !ok {
			return
		}

		var locStr string
		switch typ {
		case nordlead3.PerformanceT:
			loc, err := memory.GetPerformance(ml)
			if err != nil {
				fmt.Println("That location is either invalid or already clear.")
				return
			}
			locStr = loc.Summary()
		case nordlead3.ProgramT:
			loc, err := memory.GetProgram(ml)
			if err != nil {
				fmt.Println("That location is either invalid or already clear.")
				return
			}
			locStr = loc.Summary()
		}

		if !force {
			prompt := fmt.Sprintf("Deleting %s %d:%d : %q. Are you sure (y/N)? ", typ.String(), ml.Bank+1, ml.Location+1, locStr)

		outer:
			for {
				args := getPrompted(prompt, scanner)
				if len(args) == 0 {
					break
				}
				switch args[0] {
				case "y", "Y", "yes", "Yes", "YES":
					switch typ {
					case nordlead3.PerformanceT:
						memory.DeletePerformance(ml)
					case nordlead3.ProgramT:
						memory.DeleteProgram(ml)
					}
					fmt.Println("Baleeted!")
					return
				case "n", "N", "no", "No", "NO":
					break outer
				default:
					continue
				}
			}
		}
		fmt.Println("Ok, nevermind then!")
		return
	}
}

func export(memory *nordlead3.PatchMemory, scanner *bufio.Scanner, args []string) {
	var err error

	if tblfn, ok := getArgs(args, []string{"string", "int", "int", "string opt"}); ok {
		err = exportOne(memory, scanner, tblfn[0].(string), ml(tblfn[1].(int)-1, tblfn[2].(int)-1), tblfn[3].(string))
	} else if fn, ok := getArgs(args, []string{"perf", "all", "string opt"}); ok {
		err = exportAllPerf(memory, scanner, fn[0].(string))
	} else if fn, ok := getArgs(args, []string{"prog", "all", "string opt"}); ok {
		err = exportAllProg(memory, scanner, fn[0].(string))
	} else if bfn, ok := getArgs(args, []string{"perf", "bank", "int", "string opt"}); ok {
		err = exportPerfBank(memory, scanner, bfn[0].(int)-1, bfn[1].(string))
	} else if bfn, ok := getArgs(args, []string{"prog", "bank", "int", "string opt"}); ok {
		err = exportProgBank(memory, scanner, bfn[0].(int)-1, bfn[1].(string))
	} else {
		exportHelp()
	}

	if err != nil {
		fmt.Printf("Export error: %s\n", err)
	}
}

func exportOne(memory *nordlead3.PatchMemory, scanner *bufio.Scanner, typ string, ml nordlead3.MemoryLocation, filename string) error {
	file, err := createFile(filename, scanner)
	if err != nil {
		return err
	}
	defer file.Close()

	switch typ {
	case "prog":
		err = memory.ExportProgram(ml, file)
	case "perf":
		err = memory.ExportPerformance(ml, file)
	}
	return err
}

func exportAllPerf(memory *nordlead3.PatchMemory, scanner *bufio.Scanner, filename string) error {
	var err error

	file, err := createFile(filename, scanner)
	if err != nil {
		return err
	}
	defer file.Close()

	return memory.ExportAllPerformances(file)
}

func exportAllProg(memory *nordlead3.PatchMemory, scanner *bufio.Scanner, filename string) error {
	var err error

	file, err := createFile(filename, scanner)
	if err != nil {
		return err
	}
	defer file.Close()

	return memory.ExportAllPrograms(file)
}

func exportPerfBank(memory *nordlead3.PatchMemory, scanner *bufio.Scanner, bank int, filename string) error {
	var err error

	file, err := createFile(filename, scanner)
	if err != nil {
		return err
	}
	defer file.Close()

	return memory.ExportPerformanceBank(bank, file)
}

func exportProgBank(memory *nordlead3.PatchMemory, scanner *bufio.Scanner, bank int, filename string) error {
	var err error

	file, err := createFile(filename, scanner)
	if err != nil {
		return err
	}
	defer file.Close()

	return memory.ExportProgramBank(bank, file)
}

func createFile(filename string, scanner *bufio.Scanner) (*os.File, error) {
	var err error

	if filename == "" {
		filename, err = promptValidFilename(scanner, false)
		if err != nil {
			return nil, err
		}
	}

	// Expand ~ character first
	filename, err = homedir.Expand(filename)
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(filename)
	if !os.IsNotExist(err) {
		if err != nil {
			return nil, err
		}
		return nil, errors.New(fmt.Sprintf("Aborting: %q exists, not overwriting.\n", filename))
	}

	file, err := os.Create(filename)
	fmt.Printf("Preparing %q\n", filename)
	if err != nil {
		return nil, err
	}
	return file, nil
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

func movePrompted(memory *nordlead3.PatchMemory, scanner *bufio.Scanner, typ nordlead3.PatchType) {
	var src []nordlead3.MemoryLocation
	var dest nordlead3.MemoryLocation
	var err error

	for {
		fmt.Println("Currently selected programs to move: ", src)
		args := getPrompted(fmt.Sprintf("Choose a source location to add to the list (p: print %ss, s: sequential from last, enter: continue, a: abort)? ", typ.String()), scanner)
		if len(args) == 0 {
			break
		}
		if args[0] == "a" {
			fmt.Println("Ok, we'll try it again later!")
			return
		}
		if args[0] == "p" {
			switch typ {
			case nordlead3.PerformanceT:
				fmt.Println(memory.SprintPerformances(true))
			case nordlead3.ProgramT:
				fmt.Println(memory.SprintPrograms(true))
			}
			continue
		}
		if args[0] == "s" {
			if bl, ok := getArgs(args, []string{"string", "int", "int"}); ok {
				end := ml(bl[1].(int)-1, bl[2].(int)-1)
				if len(src) == 0 {
					fmt.Println("Cannot span without a starting location. Enter a normal location first, then try the span again.")
				} else if src[len(src)-1].Location >= end.Location {
					fmt.Println("End must be after source numerically. Spanning backwards is unsupported.")
				} else {
					if end.Bank == src[len(src)-1].Bank {
						for i := src[len(src)-1].Location + 1; i <= end.Location; i++ {
							src = append(src, ml(end.Bank, i))
						}
					} else {
						fmt.Println("Cannot span banks, sorry.")
					}
				}
			} else {
				fmt.Println("If you want a sequence of locations, enter s <bank> <location> with the position of the last location to be in the sequence.")
			}
			continue
		}
		if bl, ok := getArgs(args, []string{"int", "int"}); ok {
			src = append(src, ml(bl[0].(int)-1, bl[1].(int)-1))
		} else {
			fmt.Println(args, len(args))
			fmt.Println("I couldn't figure out what you meant, please try again with the format <bank> <location>")
		}
	}
	for {
		args := getPrompted(fmt.Sprintf("Move to which location (p to print current %ss, enter to abort)? ", typ.String()), scanner)
		if len(args) == 0 {
			fmt.Println("Ok, we'll try it again later!")
			return
		}
		if args[0] == "p" {
			switch typ {
			case nordlead3.PerformanceT:
				memory.SprintPerformances(true)
			case nordlead3.ProgramT:
				memory.SprintPrograms(true)
			}
		}
		if bl, ok := getArgs(args, []string{"int", "int"}); ok {
			dest = ml(bl[0].(int)-1, bl[1].(int)-1)
			break
		}
	}
	switch typ {
	case nordlead3.PerformanceT:
		err = memory.MovePerformances(src, dest)
	case nordlead3.ProgramT:
		err = memory.MovePrograms(src, dest)
	}
	if err != nil {
		fmt.Printf("Error moving %s: %q\n", typ.String(), err)
	} else {
		fmt.Printf("Moved!") // Todo could make a friendly summary or something.
	}
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

func usage() {
	fmt.Println("Usage: go run nl3edit <filename.syx>")
}

func help() {
	fmt.Println("Available commands are: ")
	fmt.Println(" help   | h                                              : print this help reference")
	exportHelp()
	fmt.Println(" load   | l  <filename> [<filename> ...]                 : load the requested file into memory")
	fmt.Println(" move   | m  <prog|perf>                                 : enter the move tool for programs or performances")
	fmt.Println(" rename | r  <prog|perf> <bank> <location> <new name>    : rename the indicated program or performance")
	fmt.Println(" perf        [<bank> <location>] [<depth>]               : print details of performance at that location")
	fmt.Println(" prog        [<bank> <location>] [<depth>]               : print details of program at that location")
}

func exportHelp() {
	fmt.Println(" export | e  <prog|perf> <bank> <location> [<filename>]  : export bank and location to a file")
	fmt.Println("             <prog|perf> bank <bank> [<filename>]        : export entire bank to a file")
	fmt.Println("             all <prog|perf>                             : export all progs/perfs to a file")
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

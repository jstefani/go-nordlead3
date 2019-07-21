package main

import (
	"bufio"
	"fmt"
	"github.com/malacalypse/nordlead3"
	"io"
	"os"
)

const (
	SYSEX_START = 0xF0
	SYSEX_END   = 0xF7
)

func parseSysexFile(file *os.File, memory *nordlead3.PatchMemory) (int, int, error) {
	defer file.Close()

	validFound, invalidFound := 0, 0
	reader := bufio.NewReader(file)

	fmt.Println("Beginning parsing.")

	for {
		// scan until we see an F0, we hit EOF, or an error occurs.
		_, err := reader.ReadBytes(SYSEX_START)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return 0, 0, err
			}
		}

		// Read the sysex header to see if it's data we care about
		header, _ := reader.Peek(3)
		header[1] = 0x00 // We don't care about the destination address

		// 0x33 = Clavia, 0x00 = dest. addr blanked above, 0x09 = NL3 sysex model ID
		if string(header) == string([]byte{0x33, 0x00, 0x09}) {
			programSysex, err := reader.ReadBytes(SYSEX_END)
			if err != nil {
				return 0, 0, err
			}

			sysex, err := nordlead3.ParseSysex(programSysex)
			if err == nil {
				validFound++
				memory.LoadFromSysex(sysex)
			} else {
				invalidFound++
			}
		}
	}
	fmt.Println("Finished parsing.")
	return validFound, invalidFound, nil
}

func printSummaryInfo(*nordlead3.PatchMemory) {
	fmt.Printf("Not yet implemented. :(")
}

func main() {
	if len(os.Args) == 1 {
		fmt.Println("Usage: nl3edit <filename.syx>\n")
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

	validFound, invalidFound, err := parseSysexFile(file, memory)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Found %v valid SysEx entries (%v invalid).", validFound, invalidFound)
}

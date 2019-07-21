package main

import (
	"bufio"
	"io"
	"fmt"
	"os"
	"github.com/malacalypse/nordlead3"
)

const (
	SYSEX_START = 0xF0
	SYSEX_END   = 0xF7
)


func parseSysex(file *os.File, memory *nordlead3.PatchMemory) (int, error) {
	defer file.Close()
	// returnChan := make(chan *nordlead3.NL3PatchMemory)
	// nl3PatchMemory := new(nordlead3.NL3PatchMemory)
	totalFound := 0
	reader := bufio.NewReader(file)

	// Read each sysex entity at a time
	// Message types:
	// 	0x20 : Program from Slot
	// 	0x21 : Program from Memory
	// 	0x28 : Performance from Slot
	// 	0x29 : Performance from Memory
	//
	// Proper header F0 33 XX 09 <message type> 
	//
	// Pseudocode:
	// 1. Scan until you see an F0. Complete if you reach EOF. 
	// 2. Read the F0 and the next 5 bytes.
	// 3. Validate the header against the expected value (above)
	// 4. If valid, reset the cursor at the F0 and read into the buffer until F7. If not valid, leave the cursor where it is and loop.
	// 5. Pass the entire slug off to the appropriate parser for insertion into the patch memory model.
	// 6. Loop.

	// scan until we see an F0, we hit EOF, or an error occurs. 
	fmt.Println("Beginning parsing.")

	for {
		_, err := reader.ReadBytes(SYSEX_START)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return 0, err
			}
		}

		header, _ := reader.Peek(3)
		header[1] = 0x00
		if string(header) == string([]byte{0x33, 0x00, 0x09}) {
			reader.UnreadByte()
			programSysex, err := reader.ReadBytes(SYSEX_END)
			if err != nil {
				return 0, err
			}

			nordlead3.ParseSysex(programSysex, memory)
			totalFound++
		} 
	}
	fmt.Println("Finished parsing.")
	return totalFound, nil
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

	totalFound, err := parseSysex(file, memory)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Found %v valid SysEx entries.", totalFound)
}
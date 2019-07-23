package nordlead3

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

var Uninitialized = errors.New("That location is not initialized.")

// PatchMemory holds the entire internal structure of the patch memory, including locations, names, and patch contents.
// The main object responsible for organizing programs and performances.

type PatchMemory struct {
	programs     [8][128]*ProgramLocation
	performances [2][128]*PerformanceLocation
}

// Dumps a program as sysex in NL3 format
func (memory *PatchMemory) DumpProgram(bank, location uint8) (*[]byte, error) {
	buffer := bytes.NewBuffer(nil)
	programLocation := memory.programs[bank][location]
	if programLocation == nil || programLocation.program == nil {
		return nil, Uninitialized
	}
	program := programLocation.program

	// assemble sysex prelude
	buffer.WriteString(string([]byte{0xF0, 0x33, 0x7F, 0x09, ProgramFromMemory, bank, location}))
	for i := 0; i < 16; i++ {
		currByte := programLocation.name[i]
		if uint8(currByte) < 128 {
			buffer.WriteByte(currByte)
		} else {
			panic("Sysex values cannot exceed 127!")
		}
	}
	buffer.WriteByte(programLocation.category)
	buffer.Write((*new([SpareHeaderLength]byte))[:])

	// Append version x 100 as uint16
	versionX100 := uint16(programLocation.version * 100)
	buffer.Write([]byte{byte(versionX100 >> 8), byte(versionX100)})

	// concatenate program data
	progPayload, err := program.dumpSysex()
	if err != nil {
		return nil, err
	}
	buffer.Write(*progPayload)

	// finally, bang on the trailing 0xF7
	buffer.WriteByte(0xF7)

	// grab the buffer
	sysex := buffer.Bytes()

	return &sysex, nil
}

// // Dumps a performance as sysex in NL3 format
func (memory *PatchMemory) DumpPerformance(bank, location uint8) (*[]byte, error) {
	buffer := bytes.NewBuffer(nil)
	performanceLocation := memory.performances[bank][location]
	if performanceLocation == nil || performanceLocation.performance == nil {
		return nil, Uninitialized
	}
	performance := performanceLocation.performance

	// assemble sysex prelude
	buffer.WriteString(string([]byte{0xF0, 0x33, 0x7F, 0x09, PerformanceFromMemory, bank, location}))
	for i := 0; i < 16; i++ {
		currByte := performanceLocation.name[i]
		if uint8(currByte) < 128 {
			buffer.WriteByte(currByte)
		} else {
			panic("Sysex values cannot exceed 127!")
		}
	}
	buffer.WriteByte(performanceLocation.category)
	buffer.Write((*new([SpareHeaderLength]byte))[:])

	// Append version x 100 as uint16
	versionX100 := uint16(performanceLocation.version * 100)
	buffer.Write([]byte{byte(versionX100 >> 8), byte(versionX100)})

	// concatenate performance data
	progPayload, err := performance.dumpSysex()
	if err != nil {
		return nil, err
	}
	buffer.Write(*progPayload)

	// finally, bang on the trailing 0xF7
	buffer.WriteByte(0xF7)

	// grab the buffer
	sysex := buffer.Bytes()

	return &sysex, nil
}

func (memory *PatchMemory) GetPerformance(bank, location int) (*PerformanceLocation, error) {
	loc := memory.performances[bank][location]
	if loc == nil || loc.performance == nil {
		return nil, Uninitialized
	}
	return loc, nil
}

func (memory *PatchMemory) GetProgram(bank, location int) (*ProgramLocation, error) {
	loc := memory.programs[bank][location]
	if loc == nil || loc.program == nil {
		return nil, Uninitialized
	}
	return loc, nil
}

func (memory *PatchMemory) LoadFromSysex(rawSysex []byte) error {
	err := *new(error)
	sysex, err := ParseSysex(rawSysex)
	if err != nil {
		return err
	}

	_, err = sysex.valid()
	if err != nil {
		return err
	}

	switch sysex.messageType() {
	case ProgramFromMemory, ProgramFromSlot:
		memory.loadProgramFromSysex(sysex)
	case PerformanceFromMemory, PerformanceFromSlot:
		memory.loadPerformanceFromSysex(sysex)
	}

	return nil
}

func (memory *PatchMemory) LoadFromFile(file *os.File) (numValid int, numInvalid int, err error) {
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
			sysex, err := reader.ReadBytes(SYSEX_END)
			if err != nil {
				return 0, 0, err
			}

			err = memory.LoadFromSysex(sysex)
			if err == nil {
				validFound++
			} else {
				invalidFound++
			}
		}
	}
	fmt.Println("Finished parsing.")
	return validFound, invalidFound, nil
}

func (memory *PatchMemory) loadPerformanceFromSysex(sysex *Sysex) {
	performance, err := newPerformanceFromBitstream(sysex.decodedBitstream)
	if err == nil {
		perfLocation := PerformanceLocation{name: sysex.nameAsArray(), category: sysex.category(), version: sysex.version(), performance: performance}
		memory.performances[sysex.bank()][sysex.location()] = &perfLocation
		fmt.Printf("Loaded %s: (%v:%03d) %-16.16q v%1.2f c%02x cs%02x\n", sysex.printableType(), sysex.bank(), sysex.location(), sysex.printableName(), sysex.version(), sysex.category(), sysex.checksum())
	} else {
		panic(err)
	}
}

func (memory *PatchMemory) loadProgramFromSysex(sysex *Sysex) {
	program, err := newProgramFromBitstream(sysex.decodedBitstream)
	if err == nil {
		programLocation := ProgramLocation{name: sysex.nameAsArray(), category: sysex.category(), version: sysex.version(), program: program}
		memory.programs[sysex.bank()][sysex.location()] = &programLocation
		fmt.Printf("Loaded %s: (%v:%03d) %-16.16q v%1.2f c%02x cs%02x\n", sysex.printableType(), sysex.bank(), sysex.location(), sysex.printableName(), sysex.version(), sysex.category(), sysex.checksum())
	} else {
		panic(err)
	}
}

func (memory *PatchMemory) PrintPrograms(omitBlank bool) string {
	var result []string

	result = append(result, "\n***** PROGRAMS ******\n")
	for numBank, bank := range memory.programs {
		bank_header := fmt.Sprintf("\n*** Bank %v ***\n", numBank+1)
		result = append(result, bank_header)

		for location, program := range bank {
			if program != nil || !omitBlank {
				result = append(result, fmt.Sprintf("   %3d : %s", location, program.Summary()))
			}
		}

	}

	return strings.Join(result, "\n")
}

func (memory *PatchMemory) PrintPerformances(omitBlank bool) string {
	var result []string

	result = append(result, "\n***** PERFORMANCES ******\n")

	for numBank, bank := range memory.performances {
		bank_header := fmt.Sprintf("\n*** Bank %v ***\n", numBank+1)
		result = append(result, bank_header)

		for location, performance := range bank {
			if performance != nil || !omitBlank {
				result = append(result, fmt.Sprintf("   %3d : %s", location, performance.Summary()))
			}
		}
	}

	return strings.Join(result, "\n")
}

func newProgramFromBitstream(data []byte) (*Program, error) {
	program := new(Program)
	err := populateStructFromBitstream(program, data)
	return program, err
}

func newPerformanceFromBitstream(data []byte) (*Performance, error) {
	performance := new(Performance)
	err := populateStructFromBitstream(performance, data)
	return performance, err
}

package nordlead3

import (
	"bufio"
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
	programs     [8][128]*Program
	performances [2][128]*Performance
}

// Dumps a program as sysex in NL3 format
func (memory *PatchMemory) DumpProgram(bank, location uint8) (*[]byte, error) {
	program, err := memory.GetProgram(bank, location)
	if err != nil {
		return nil, err
	}

	sysex, err := toSysex(program, bank, location)
	if err != nil {
		return nil, err
	}

	return sysex, nil
}

func (memory *PatchMemory) DumpPerformances() (*[]byte, error) {
	var output []byte

	for bank, performances := range memory.performances {
		for location, _ := range performances {
			perfdata, err := memory.DumpProgram(uint8(bank), uint8(location))
			if err != nil {
				return nil, err
			}
			output = append(output, *perfdata...)
		}
	}
	return &output, nil
}

func (memory *PatchMemory) DumpPrograms() (*[]byte, error) {
	var output []byte

	for bank, programs := range memory.programs {
		for location, _ := range programs {
			programdata, err := memory.DumpProgram(uint8(bank), uint8(location))
			if err != nil {
				return nil, err
			}
			output = append(output, *programdata...)
		}
	}
	return &output, nil
}

// // Dumps a performance as sysex in NL3 format
func (memory *PatchMemory) DumpPerformance(bank, location uint8) (*[]byte, error) {
	performance, err := memory.GetPerformance(bank, location)
	if err != nil {
		return nil, err
	}

	sysex, err := toSysex(performance, bank, location)
	if err != nil {
		return nil, err
	}

	return sysex, nil
}

func (memory *PatchMemory) GetPerformance(bank, location uint8) (*Performance, error) {
	loc := memory.performances[bank][location]
	if loc == nil || loc.data == nil {
		return nil, Uninitialized
	}
	return loc, nil
}

func (memory *PatchMemory) GetProgram(bank, location uint8) (*Program, error) {
	loc := memory.programs[bank][location]
	if loc == nil || loc.data == nil {
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
	validFound, invalidFound := 0, 0
	reader := bufio.NewReader(file)

	for {
		// scan until we see an F0, we hit EOF, or an error occurs.
		_, err := reader.ReadBytes(SYSEX_START)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return validFound, invalidFound, err
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
	return validFound, invalidFound, nil
}

func (memory *PatchMemory) loadPerformanceFromSysex(sysex *Sysex) {
	performanceData, err := newPerformanceFromBitstream(sysex.decodedBitstream)
	if err == nil {
		performance := Performance{name: sysex.nameAsArray(), category: sysex.category(), version: sysex.version(), data: performanceData}
		if existing, err := memory.GetPerformance(sysex.bank(), sysex.location()); err == nil {
			fmt.Printf("Overwriting %d:%d %q with %q\n", sysex.bank(), sysex.location(), existing.PrintableName(), sysex.printableName())
		}
		memory.performances[sysex.bank()][sysex.location()] = &performance
		// fmt.Printf("Loaded %s: (%v:%03d) %-16.16q v%1.2f c%02x cs%02x\n", sysex.printableType(), sysex.bank(), sysex.location(), sysex.printableName(), sysex.version(), sysex.category(), sysex.checksum())
	} else if err == io.EOF {
		fmt.Println("An EOF error occurred during import. The data may not have been in the expected format.")
	} else {
		panic(err)
	}
}

func (memory *PatchMemory) loadProgramFromSysex(sysex *Sysex) {
	programData, err := newProgramFromBitstream(sysex.decodedBitstream)
	if err == nil {
		program := Program{name: sysex.nameAsArray(), category: sysex.category(), version: sysex.version(), data: programData}
		// detect overwrite
		if existing, err := memory.GetProgram(sysex.bank(), sysex.location()); err == nil {
			fmt.Printf("Overwriting %d:%d %q with %q\n", sysex.bank(), sysex.location(), existing.PrintableName(), sysex.printableName())
		}
		memory.programs[sysex.bank()][sysex.location()] = &program
		// fmt.Printf("Loaded %s: (%v:%03d) %-16.16q v%1.2f c%02x cs%02x\n", sysex.printableType(), sysex.bank(), sysex.location(), sysex.printableName(), sysex.version(), sysex.category(), sysex.checksum())
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

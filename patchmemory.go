package nordlead3

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// PatchMemory holds the entire internal structure of the patch memory, including locations, names, and patch contents.
// The main object responsible for organizing programs and performances.

type PatchMemory struct {
	programs     [8][128]*Program
	performances [2][128]*Performance
}

type patchType int

const (
	patchProgram = iota
	patchPerformance
)

type patchLocation struct {
	patchType patchType
	bank      uint8
	position  uint8
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

// Accepts an array of patchLocations and exports them to the same file
func (memory *PatchMemory) exportLocations(locations []patchLocation, filename string) error {
	var (
		exportdata []byte
		err        error
		fdata      *[]byte
	)

	for _, location := range locations {
		switch location.patchType {
		case patchProgram:
			fdata, err = memory.DumpProgram(location.bank, location.position)
		case patchPerformance:
			fdata, err = memory.DumpPerformance(location.bank, location.position)
		default:
			// skip
		}

		if err == ErrorUninitialized {
			continue
		} else if err != nil {
			return err
		}

		exportdata = append(exportdata, *fdata...)
	}

	if len(exportdata) == 0 {
		return ErrorNoDataToWrite
	}
	return exportToFile(&exportdata, filename, false)
}

func (memory *PatchMemory) ExportAllPerformances(filename string) error {
	var locations []patchLocation

	for bank, performances := range memory.performances {
		for position, _ := range performances {
			locations = append(locations, patchLocation{patchPerformance, uint8(bank), uint8(position)})
		}
	}
	return memory.exportLocations(locations, filename)
}

func (memory *PatchMemory) ExportAllPrograms(filename string) error {
	var locations []patchLocation

	for bank, programs := range memory.programs {
		for position, _ := range programs {
			locations = append(locations, patchLocation{patchProgram, uint8(bank), uint8(position)})
		}
	}
	return memory.exportLocations(locations, filename)
}

func (memory *PatchMemory) ExportPerformance(bank, location uint8, filename string) error {
	locations := []patchLocation{patchLocation{patchPerformance, bank, location}}
	return memory.exportLocations(locations, filename)
}

func (memory *PatchMemory) ExportPerformanceBank(bank uint8, filename string) error {
	var locations []patchLocation

	for location, _ := range memory.performances[bank] {
		locations = append(locations, patchLocation{patchPerformance, bank, uint8(location)})
	}
	return memory.exportLocations(locations, filename)
}

func (memory *PatchMemory) ExportProgram(bank, location uint8, filename string) error {
	locations := []patchLocation{patchLocation{patchProgram, bank, location}}
	return memory.exportLocations(locations, filename)
}

func (memory *PatchMemory) ExportProgramBank(bank uint8, filename string) error {
	var locations []patchLocation

	for location, _ := range memory.programs[bank] {
		locations = append(locations, patchLocation{patchProgram, bank, uint8(location)})
	}
	return memory.exportLocations(locations, filename)
}

func (memory *PatchMemory) GetPerformance(bank, location uint8) (*Performance, error) {
	loc := memory.performances[bank][location]
	if loc == nil || loc.data == nil {
		return nil, ErrorUninitialized
	}
	return loc, nil
}

func (memory *PatchMemory) GetProgram(bank, location uint8) (*Program, error) {
	loc := memory.programs[bank][location]
	if loc == nil || loc.data == nil {
		return nil, ErrorUninitialized
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

// returns an error if any of the len(src) locations following dest are not empty, or if src contains
// patchLocations of different patchTypes
// todo: could probably do this with a state concept in the patch memory too, but that's for later
//       e.g. create a new patchmemory clone of the current one, start replacing, and if we hit a non-nil dest, abort
//            if we don't, swap the new state for the old state as the current valid state of the memory.
//            bonus is that we can store the old state as an undo point.
func (memory *PatchMemory) move(src []patchLocation, dest patchLocation) error {
	var err error
	var moved []patchLocation
	tloc := src[0].patchType

	for i, _ := range src {
		u := uint8(i)
		if dest.position+u > 127 {
			memory.move(moved, src[0]) // undo the ones moved so far
			return errors.New("Not enough room in that bank")
		}

		currDest := patchLocation{tloc, dest.bank, dest.position + u}
		switch tloc {
		case patchPerformance:
			err = memory.movePerformance(src[i], currDest)
		case patchProgram:
			err = memory.moveProgram(src[i], currDest)
		}

		if err != nil { // currDest was not overwritten
			memory.move(moved, src[0])
			return err
		} else {
			moved = append(moved, currDest)
		}
	}

	return err
}

func (memory *PatchMemory) movePerformance(src patchLocation, dest patchLocation) error {
	if src.patchType != patchPerformance || dest.patchType != patchPerformance {
		return errors.New("Cannot move different types of patches")
	}
	_, err := memory.GetPerformance(dest.bank, dest.position)
	if err != ErrorUninitialized {
		return errors.New("Destination is not empty")
	}
	memory.performances[dest.bank][dest.position] = memory.performances[src.bank][src.position]
	memory.performances[src.bank][src.position] = nil
	return nil
}

func (memory *PatchMemory) moveProgram(src patchLocation, dest patchLocation) error {
	if src.patchType != patchProgram || dest.patchType != patchProgram {
		return errors.New("Cannot move different types of patches")
	}
	_, err := memory.GetProgram(dest.bank, dest.position)
	if err != ErrorUninitialized {
		return errors.New("Destination is not empty")
	}
	memory.programs[dest.bank][dest.position] = memory.programs[src.bank][src.position]
	memory.programs[src.bank][src.position] = nil
	return nil
}

func (memory *PatchMemory) PrintPrograms(omitBlank bool) string {
	var result []string

	result = append(result, "\n***** PROGRAMS ******\n")
	for numBank, bank := range memory.programs {
		bank_header := fmt.Sprintf("\n*** Bank %v ***\n", numBank+1)
		result = append(result, bank_header)

		for location, program := range bank {
			if program != nil || !omitBlank {
				result = append(result, fmt.Sprintf("   %3d : %s", location+1, program.Summary()))
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
				result = append(result, fmt.Sprintf("   %3d : %s", location+1, performance.Summary()))
			}
		}
	}

	return strings.Join(result, "\n")
}

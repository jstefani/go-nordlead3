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

const (
	programT = iota
	performanceT
)

const (
	bankSize     = 128
	numPerfBanks = 2
	numProgBanks = 8
)

type PatchMemory struct {
	programs     [numProgBanks * bankSize]*Program
	performances [numPerfBanks * bankSize]*Performance
}

type patchType int

type patchRef struct {
	patchType patchType
	index     int
}

func (ref *patchRef) bank() int {
	return bank(ref.index)
}

func (ref *patchRef) location() int {
	return location(ref.index)
}

func (ref *patchRef) valid() bool {
	return valid(ref.patchType, ref.index)
}

// Dumps a program as sysex in NL3 format
func (memory *PatchMemory) DumpProgram(bank, location int) (*[]byte, error) {
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

func (memory *PatchMemory) DumpPrograms() (*[]byte, error) {
	var output []byte

	for i, _ := range memory.programs {
		programdata, err := memory.DumpProgram(bankloc(i))
		if err != nil {
			return nil, err
		}
		output = append(output, *programdata...)
	}
	return &output, nil
}

// // Dumps a performance as sysex in NL3 format
func (memory *PatchMemory) DumpPerformance(bank, location int) (*[]byte, error) {
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

func (memory *PatchMemory) DumpPerformances() (*[]byte, error) {
	var output []byte

	for i, _ := range memory.performances {
		perfdata, err := memory.DumpPerformance(bankloc(i))
		if err != nil {
			return nil, err
		}
		output = append(output, *perfdata...)
	}
	return &output, nil
}

// Accepts an array of patchLocations and exports them to the same file
func (memory *PatchMemory) exportLocations(refs []patchRef, filename string) error {
	var (
		exportdata []byte
		err        error
		fdata      *[]byte
	)

	for _, ref := range refs {
		switch ref.patchType {
		case programT:
			fdata, err = memory.DumpProgram(ref.bank(), ref.location())
		case performanceT:
			fdata, err = memory.DumpPerformance(ref.bank(), ref.location())
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
	var refs []patchRef

	for i, _ := range memory.performances {
		refs = append(refs, patchRef{performanceT, i})
	}
	return memory.exportLocations(refs, filename)
}

func (memory *PatchMemory) ExportAllPrograms(filename string) error {
	var refs []patchRef

	for i, _ := range memory.programs {
		refs = append(refs, patchRef{programT, i})
	}
	return memory.exportLocations(refs, filename)
}

func (memory *PatchMemory) ExportPerformance(bank, location int, filename string) error {
	refs := []patchRef{patchRef{performanceT, index(bank, location)}}
	return memory.exportLocations(refs, filename)
}

func (memory *PatchMemory) ExportPerformanceBank(bank int, filename string) error {
	var refs []patchRef

	for i := 0; i < bankSize; i++ {
		refs = append(refs, patchRef{performanceT, index(bank, i)})
	}
	return memory.exportLocations(refs, filename)
}

func (memory *PatchMemory) ExportProgram(bank, location int, filename string) error {
	refs := []patchRef{patchRef{programT, index(bank, location)}}
	return memory.exportLocations(refs, filename)
}

func (memory *PatchMemory) ExportProgramBank(bank int, filename string) error {
	var refs []patchRef

	for i := 0; i < bankSize; i++ {
		refs = append(refs, patchRef{programT, index(bank, i)})
	}
	return memory.exportLocations(refs, filename)
}

func (memory *PatchMemory) GetPerformance(bank, location int) (*Performance, error) {
	if index, valid := indexv(performanceT, bank, location); valid {
		if memory.initialized(performanceT, index) {
			return memory.performances[index], nil
		}
	}
	return nil, ErrorUninitialized
}

func (memory *PatchMemory) GetProgram(bank, location int) (*Program, error) {
	if index, valid := indexv(programT, bank, location); valid {
		if memory.initialized(programT, index) {
			return memory.programs[index], nil
		}
	}
	return nil, ErrorUninitialized
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
		memory.performances[index(sysex.bank(), sysex.location())] = &performance
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
		memory.programs[index(sysex.bank(), sysex.location())] = &program
		// fmt.Printf("Loaded %s: (%v:%03d - %d) %-16.16q v%1.2f c%02x cs%02x\n", sysex.printableType(), sysex.bank(), sysex.location(), index(sysex.bank(), sysex.location()), sysex.printableName(), sysex.version(), sysex.category(), sysex.checksum())
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
func (memory *PatchMemory) move(src []patchRef, dest patchRef) error {
	var err error
	var moved []patchRef

	if len(src) == 0 {
		return nil
	}

	refT := src[0].patchType

	for i, _ := range src {
		if !valid(refT, dest.index+i) {
			memory.move(moved, src[0]) // undo the ones moved so far
			return errors.New("Not enough room in that bank")
		}

		currDest := patchRef{refT, dest.index + i}
		switch refT {
		case performanceT:
			err = memory.movePerformance(src[i], currDest)
		case programT:
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

func (memory *PatchMemory) movePerformance(src patchRef, dest patchRef) error {
	if src.patchType != performanceT || dest.patchType != performanceT {
		return errors.New("Cannot move different types of patches")
	}
	_, err := memory.GetPerformance(dest.bank(), dest.location())
	if err != ErrorUninitialized {
		return errors.New("Destination is not empty")
	}
	memory.performances[dest.index] = memory.performances[src.index]
	memory.performances[src.index] = nil
	return nil
}

func (memory *PatchMemory) moveProgram(src patchRef, dest patchRef) error {
	if src.patchType != programT || dest.patchType != programT {
		return errors.New("Cannot move different types of patches")
	}
	_, err := memory.GetProgram(dest.bank(), dest.location())
	if err != ErrorUninitialized {
		return errors.New("Destination is not empty")
	}
	memory.programs[dest.index] = memory.programs[src.index]
	memory.programs[src.index] = nil
	return nil
}

func (memory *PatchMemory) PrintPrograms(omitBlank bool) string {
	var result []string

	result = append(result, "\n***** PROGRAMS ******\n")
	for i, program := range memory.programs {
		bank, location := bankloc(i)
		bank_header := fmt.Sprintf("\n*** Bank %v ***\n", bank+1)
		result = append(result, bank_header)

		if memory.initialized(programT, i) || !omitBlank {
			result = append(result, fmt.Sprintf("   %3d : %s", location+1, program.Summary()))
		}
	}

	return strings.Join(result, "\n")
}

func (memory *PatchMemory) PrintPerformances(omitBlank bool) string {
	var result []string

	result = append(result, "\n***** PERFORMANCES ******\n")

	for i, perf := range memory.performances {
		bank, location := bankloc(i)
		bank_header := fmt.Sprintf("\n*** Bank %v ***\n", bank+1)
		result = append(result, bank_header)

		if memory.initialized(performanceT, i) || !omitBlank {
			result = append(result, fmt.Sprintf("   %3d : %s", location+1, perf.Summary()))
		}
	}

	return strings.Join(result, "\n")
}

func bank(index int) int {
	return index / bankSize
}

func bankv(pt patchType, index int) (int, bool) {
	return index / bankSize, valid(pt, index)
}

func location(index int) int {
	return index % bankSize
}

func locationv(pt patchType, index int) (int, bool) {
	return index % bankSize, valid(pt, index)
}

func index(bank, location int) int {
	return bank*bankSize + location
}

func indexv(pt patchType, bank, location int) (int, bool) {
	index := bank*bankSize + location
	return index, valid(pt, index)
}

// Useful when we know the location is valid already
func bankloc(index int) (int, int) {
	return bank(index), location(index)
}

func valid(pt patchType, index int) bool {
	var numBanks int

	switch pt {
	case performanceT:
		numBanks = numPerfBanks
	case programT:
		numBanks = numProgBanks
	default:
		// skip
	}
	return index >= 0 && index < numBanks*bankSize
}

func (memory *PatchMemory) initialized(pt patchType, index int) (result bool) {
	if !valid(pt, index) {
		return
	}

	switch pt {
	case performanceT:
		result = memory.performances[index] != nil
	case programT:
		result = memory.programs[index] != nil
	default:
		result = false
	}

	return
}

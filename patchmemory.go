package nordlead3

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	bankSize     = 128
	numPerfBanks = 2
	numProgBanks = 8
)

// transferModes
const (
	copyM = true
	moveM = false
)

// definition
var performanceSlotRef = patchRef{PerformanceT, SlotT, 0}

type transferMode bool

type PatchMemory struct {
	performances    [numPerfBanks * bankSize]*Performance
	programs        [numProgBanks * bankSize]*Program
	slotPerformance *Performance
	slotPrograms    [4]*Program
}

func (memory *PatchMemory) clear(ref patchRef) {
	if memory.initialized(ref) {
		switch ref.PatchType {
		case PerformanceT:
			*memory.perfPtr(ref) = nil
		case ProgramT:
			*memory.progPtr(ref) = nil
		}
	}
}

// Formats a patch as sysex in NL3 format
func (memory *PatchMemory) export(ref patchRef) (*[]byte, error) {
	patch, err := memory.Get(ref)
	if err != nil {
		return nil, err
	}
	if sysexable, ok := patch.(sysexable); ok {
		sysex, err := toSysex(sysexable, ref)
		if err != nil {
			return nil, err
		}

		return sysex, nil
	}
	return nil, errors.New("Requested location cannot be formatted as sysex.")
}

// Accepts an array of patchLocations and exports them to the same file
func (memory *PatchMemory) exportLocations(refs []patchRef, filename string) error {
	var (
		exportdata []byte
	)

	for _, ref := range refs {
		fdata, err := memory.export(patchRef{PerformanceT, MemoryT, ref.index})
		if err != nil {
			return err
		}

		if err == ErrUninitialized {
			continue // skip silently
		} else if err != nil {
			return err
		}

		exportdata = append(exportdata, *fdata...)
	}

	if len(exportdata) == 0 {
		return ErrNoDataToWrite
	}
	return exportToFile(&exportdata, filename, false)
}

func (memory *PatchMemory) ExportAllPerformances(filename string) error {
	var refs []patchRef

	for i, _ := range memory.performances {
		refs = append(refs, patchRef{PerformanceT, MemoryT, i})
	}
	return memory.exportLocations(refs, filename)
}

func (memory *PatchMemory) ExportAllPrograms(filename string) error {
	var refs []patchRef

	for i, _ := range memory.programs {
		refs = append(refs, patchRef{ProgramT, MemoryT, i})
	}
	return memory.exportLocations(refs, filename)
}

func (memory *PatchMemory) ExportPerformance(bank, location int, filename string) error {
	refs := []patchRef{patchRef{PerformanceT, MemoryT, index(bank, location)}}
	return memory.exportLocations(refs, filename)
}

func (memory *PatchMemory) ExportPerformanceBank(bank int, filename string) error {
	var refs []patchRef

	for i := 0; i < bankSize; i++ {
		refs = append(refs, patchRef{PerformanceT, MemoryT, index(bank, i)})
	}
	return memory.exportLocations(refs, filename)
}

func (memory *PatchMemory) ExportProgram(bank, location int, filename string) error {
	refs := []patchRef{patchRef{ProgramT, MemoryT, index(bank, location)}}
	return memory.exportLocations(refs, filename)
}

func (memory *PatchMemory) ExportProgramBank(bank int, filename string) error {
	var refs []patchRef

	for i := 0; i < bankSize; i++ {
		refs = append(refs, patchRef{ProgramT, MemoryT, index(bank, i)})
	}
	return memory.exportLocations(refs, filename)
}

// Returns a generic patch interface, could be either a program or a performance.
// It remains to the receiver to assert the patch to the necessary type.
func (memory *PatchMemory) Get(ref patchRef) (patch, error) {
	var result patch

	if !ref.valid() {
		return nil, ErrInvalidLocation
	}
	if !memory.initialized(ref) {
		return nil, ErrUninitialized
	}
	switch ref.PatchType {
	case PerformanceT:
		result = *memory.perfPtr(ref)
	case ProgramT:
		result = *memory.progPtr(ref)
	}

	return result, nil
}

// Force sets the location in ref to the patch pointer, cast appropriately.
// Does not care if the location is already occupied (current contents will be lost if not previously copied to another location)
// Returns an error if the patch and ref are not the same type.
func (memory *PatchMemory) set(ref patchRef, patch patch) error {
	err := ErrInvalidLocation

	switch ref.PatchType {
	case PerformanceT:
		if performancePtr, ok := patch.(*Performance); ok {
			*memory.perfPtr(ref) = performancePtr
			err = nil
		}
	case ProgramT:
		if programPtr, ok := patch.(*Program); ok {
			*memory.progPtr(ref) = programPtr
			err = nil
		}
	}
	return err
}

func (memory *PatchMemory) Import(rawSysex []byte) error {
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

func (memory *PatchMemory) ImportFrom(input io.Reader) (numValid int, numInvalid int, err error) {
	validFound, invalidFound := 0, 0
	reader := bufio.NewReader(input)

	// TODO: Refactor this as a scanner break function and scan the string elegantly
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

			err = memory.Import(sysex)
			if err == nil {
				validFound++
			} else {
				invalidFound++
			}
		}
	}
	return validFound, invalidFound, nil
}

func (memory *PatchMemory) loadPerformanceFromSysex(sysex *Sysex) error {
	var ref patchRef

	performanceData, err := newPerformanceFromBitstream(sysex.decodedBitstream)
	if err == nil {
		performance := Performance{
			name:     sysex.nameAsArray(),
			category: sysex.category(),
			version:  sysex.version(),
			data:     performanceData,
		}

		if sysex.messageType() == PerformanceFromSlot {
			ref = performanceSlotRef
		} else {
			ref = patchRef{PerformanceT, MemoryT, index(sysex.bank(), sysex.location())}
		}
		if existing, err := memory.Get(ref); err == nil {
			fmt.Printf("Overwriting %s (%q) with %q\n", ref.String(), existing.PrintableName(), sysex.printableName())
		}
		err = memory.set(ref, &performance)
	}
	return err
}

func (memory *PatchMemory) loadProgramFromSysex(sysex *Sysex) error {
	var ref patchRef

	programData, err := newProgramFromBitstream(sysex.decodedBitstream)
	if err == nil {
		program := Program{
			name:     sysex.nameAsArray(),
			category: sysex.category(),
			version:  sysex.version(),
			data:     programData,
		}
		if sysex.messageType() == ProgramFromSlot {
			ref = patchRef{ProgramT, SlotT, sysex.bank()}
		} else {
			ref = patchRef{ProgramT, MemoryT, index(sysex.bank(), sysex.location())}
		}
		if existing, err := memory.Get(ref); err == nil {
			fmt.Printf("Overwriting %s (%q) with %q\n", ref.String(), existing.PrintableName(), sysex.printableName())
		}
		err = memory.set(ref, &program)
	}
	return err
}

// Transfer can behave as a copy (mode is copyM) or a move (mode is moveM).
// returns an error if any of the len(src) locations following dest are not empty, or if src contains
// patchLocations of different patchTypes
// todo: could probably do this with a state concept in the patch memory too, but that's for later
//       e.g. create a new patchmemory clone of the current one, start replacing, and if we hit a non-nil dest, abort
//            if we don't, swap the new state for the old state as the current valid state of the memory.
//            bonus is that we can store the old state as an undo point.
func (memory *PatchMemory) Transfer(src []patchRef, dest patchRef, mode transferMode) error {
	var err error
	var moved []patchRef

	if len(src) == 0 {
		return nil
	}
	if src[0].PatchType != dest.PatchType {
		return ErrXferTypeMismatch
	}

	for i, currSrc := range src {
		currDest := patchRef{dest.PatchType, dest.source, dest.index + i}

		if !currDest.valid() {
			err = ErrMemoryOverflow
		}
		if currSrc.PatchType != currDest.PatchType {
			err = ErrXferTypeMismatch
		}
		if memory.initialized(currDest) {
			err = ErrMemoryOccupied
		}
		if err != nil {
			memory.Transfer(moved, src[0], moveM) // undo the ones moved so far
			break
		}

		// Handle move of each program type since their pointer values are separate
		switch currSrc.PatchType {
		case PerformanceT:
			srcPtr := memory.perfPtr(currSrc)
			destPtr := memory.perfPtr(currDest)

			*destPtr = *srcPtr
			if mode == moveM {
				*srcPtr = nil
			}
		case ProgramT:
			srcPtr := memory.progPtr(currSrc)
			destPtr := memory.progPtr(currDest)

			*destPtr = *srcPtr
			if mode == moveM {
				*srcPtr = nil
			}
		}
		moved = append(moved, currDest)
	}

	return err
}

func (memory *PatchMemory) SprintPrograms(omitBlank bool) string {
	var result []string
	currBank := -1 // won't match any bank

	result = append(result, "\n***** PROGRAMS ******\n")
	for i, program := range memory.programs {
		bank, location := bankloc(i)

		if memory.initialized(patchRef{ProgramT, MemoryT, i}) || !omitBlank {
			if bank != currBank {
				bank_header := fmt.Sprintf("\n*** Bank %v ***\n", bank+1)
				result = append(result, bank_header)
				currBank = bank
			}

			result = append(result, fmt.Sprintf("   %3d : %s", location+1, program.Summary()))
		}
	}

	return strings.Join(result, "\n")
}

func (memory *PatchMemory) SprintPerformances(omitBlank bool) string {
	var result []string
	currBank := -1 // won't match any bank

	result = append(result, "\n***** PERFORMANCES ******\n")

	for i, perf := range memory.performances {
		bank, location := bankloc(i)

		if memory.initialized(patchRef{PerformanceT, MemoryT, i}) || !omitBlank {
			if bank != currBank {
				bank_header := fmt.Sprintf("\n*** Bank %v ***\n", bank+1)
				result = append(result, bank_header)
				currBank = bank
			}

			result = append(result, fmt.Sprintf("   %3d : %s", location+1, perf.Summary()))
		}
	}

	return strings.Join(result, "\n")
}

func (memory *PatchMemory) initialized(ref patchRef) (result bool) {
	if !ref.valid() {
		return
	}

	switch ref.source {
	case SlotT:
		switch ref.PatchType {
		case PerformanceT:
			result = memory.slotPerformance != nil
		case ProgramT:
			result = memory.slotPrograms[ref.index] != nil
		}
	case MemoryT:
		switch ref.PatchType {
		case PerformanceT:
			result = memory.performances[ref.index] != nil
		case ProgramT:
			result = memory.programs[ref.index] != nil
		}
	}

	return
}

// panics if given an invalid patchRef
func (memory *PatchMemory) perfPtr(ref patchRef) (perf **Performance) {
	if ref.PatchType != PerformanceT || !ref.valid() {
		panic("Invalid reference, cannot return pointer!")
	}

	switch ref.source {
	case SlotT:
		perf = &memory.slotPerformance
	case MemoryT:
		perf = &memory.performances[ref.index]
	}
	return
}

// panics if given an invalid patchRef
func (memory *PatchMemory) progPtr(ref patchRef) (prog **Program) {
	if ref.PatchType != ProgramT || !ref.valid() {
		panic("Invalid reference, cannot return pointer!")
	}

	switch ref.source {
	case SlotT:
		prog = &memory.slotPrograms[ref.index]
	case MemoryT:
		prog = &memory.programs[ref.index]
	}
	return
}

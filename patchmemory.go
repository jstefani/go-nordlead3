package nordlead3

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	BankSize            = 128
	NumPerformanceBanks = 2
	NumProgramBanks     = 8
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
	performances    [NumPerformanceBanks * BankSize]*Performance
	programs        [NumProgramBanks * BankSize]*Program
	slotPerformance *Performance
	slotPrograms    [4]*Program
}

func (memory *PatchMemory) CopyPerformanceToSlot(ml MemoryLocation) error {
	src := patchRef{PerformanceT, MemoryT, ml.index()}
	dest := performanceSlotRef
	return memory.copy(src, dest)
}

func (memory *PatchMemory) CopyProgramToSlot(ml MemoryLocation, index int) error {
	src := patchRef{ProgramT, MemoryT, ml.index()}
	dest := patchRef{ProgramT, SlotT, index}
	return memory.copy(src, dest)
}

func (memory *PatchMemory) CopySlotToPerformance(ml MemoryLocation) error {
	src := performanceSlotRef
	dest := patchRef{PerformanceT, MemoryT, ml.index()}
	return memory.copy(src, dest)
}

func (memory *PatchMemory) CopySlotToProgram(index int, ml MemoryLocation) error {
	src := patchRef{ProgramT, SlotT, index}
	dest := patchRef{ProgramT, MemoryT, ml.index()}
	return memory.copy(src, dest)
}

func (memory *PatchMemory) DeletePerformance(ml MemoryLocation) {
	ref := patchRef{PerformanceT, MemoryT, ml.index()}
	memory.clear(ref)
}

func (memory *PatchMemory) DeleteProgram(ml MemoryLocation) {
	ref := patchRef{ProgramT, MemoryT, ml.index()}
	memory.clear(ref)
}

func (memory *PatchMemory) ExportAllPerformances(writer io.Writer) error {
	var refs []patchRef

	for i, _ := range memory.performances {
		refs = append(refs, patchRef{PerformanceT, MemoryT, i})
	}
	return memory.exportLocations(refs, writer)
}

func (memory *PatchMemory) ExportAllPrograms(writer io.Writer) error {
	var refs []patchRef

	for i, _ := range memory.programs {
		refs = append(refs, patchRef{ProgramT, MemoryT, i})
	}
	return memory.exportLocations(refs, writer)
}

func (memory *PatchMemory) ExportPerformance(ml MemoryLocation, writer io.Writer) error {
	refs := []patchRef{patchRef{PerformanceT, MemoryT, ml.index()}}
	return memory.exportLocations(refs, writer)
}

func (memory *PatchMemory) ExportPerformanceBank(bank int, writer io.Writer) error {
	var refs []patchRef

	for i := 0; i < BankSize; i++ {
		refs = append(refs, patchRef{PerformanceT, MemoryT, index(bank, i)})
	}
	return memory.exportLocations(refs, writer)
}

func (memory *PatchMemory) ExportPerformanceSlot(writer io.Writer) error {
	return memory.exportLocations([]patchRef{performanceSlotRef}, writer)
}

func (memory *PatchMemory) ExportProgram(ml MemoryLocation, writer io.Writer) error {
	refs := []patchRef{patchRef{ProgramT, MemoryT, ml.index()}}
	return memory.exportLocations(refs, writer)
}

func (memory *PatchMemory) ExportProgramBank(bank int, writer io.Writer) error {
	var refs []patchRef

	for i := 0; i < BankSize; i++ {
		refs = append(refs, patchRef{ProgramT, MemoryT, index(bank, i)})
	}
	return memory.exportLocations(refs, writer)
}

func (memory *PatchMemory) ExportProgramSlot(slot int, writer io.Writer) error {
	return memory.exportLocations([]patchRef{patchRef{ProgramT, SlotT, slot}}, writer)
}

// Straight import: try to load into patch memory the way the file was dumped out, preserving
// locations from the sysex.
func (memory *PatchMemory) Import(input io.Reader, overwrite bool) (numValid int, numInvalid int, err error) {
	validFound, invalidFound := 0, 0
	scanner := bufio.NewScanner(input)
	scanner.Split(splitSysex(vendorNord, modelNL3))

	for ok := scanner.Scan(); ; ok = scanner.Scan() {
		if !ok {
			return validFound, invalidFound, scanner.Err()
		}
		sysex, err := parseSysex(scanner.Bytes())
		if err != nil {
			invalidFound++
			continue
		}

		err = memory.importTo(sysex, sysex.toPatchRef(), overwrite)
		if err == nil {
			validFound++
		} else {
			invalidFound++
		}
	}
	return validFound, invalidFound, nil
}

// Custom import: loads the patches of the designated type only, starting at the memory location given.
// Subsequent patches found in the same datastream will populate subsequent data locations.
// Data in memory can be lost if overwrite is set to true and there are loaded patches in the locations populated by the import.
func (memory *PatchMemory) ImportTo(input io.Reader, pt PatchType, ml MemoryLocation, overwrite bool) (numImported, numRejected int, err error) {
	validFound, invalidFound := 0, 0
	dest := patchRef{pt, MemoryT, ml.index()}

	scanner := bufio.NewScanner(input)
	scanner.Split(splitSysex(vendorNord, modelNL3))

	for ok := scanner.Scan(); ; ok = scanner.Scan() {
		if !ok {
			return validFound, invalidFound, scanner.Err()
		}
		sysex, err := parseSysex(scanner.Bytes())
		if err != nil {
			invalidFound++
			continue
		}

		err = memory.importTo(sysex, dest, overwrite)
		if err == nil {
			validFound++
			dest = dest.increment()
		} else {
			invalidFound++
		}
	}
	return validFound, invalidFound, nil
}

func (memory *PatchMemory) GetPerformance(ml MemoryLocation) (*Performance, error) {
	ref := patchRef{PerformanceT, MemoryT, ml.index()}
	patch, err := memory.get(ref)
	if err != nil {
		return nil, err
	}
	return patch.(*Performance), nil
}

func (memory *PatchMemory) GetProgram(ml MemoryLocation) (*Program, error) {
	ref := patchRef{ProgramT, MemoryT, ml.index()}
	patch, err := memory.get(ref)
	if err != nil {
		return nil, err
	}
	return patch.(*Program), nil
}

func (memory *PatchMemory) GetSlotPerformance() (*Performance, error) {
	ref := performanceSlotRef
	patch, err := memory.get(ref)
	if err != nil {
		return nil, err
	}
	return patch.(*Performance), nil
}

func (memory *PatchMemory) GetSlotProgram(index int) (*Program, error) {
	ref := patchRef{ProgramT, SlotT, index}
	patch, err := memory.get(ref)
	if err != nil {
		return nil, err
	}
	return patch.(*Program), nil
}

func (memory *PatchMemory) MovePerformances(src []MemoryLocation, dest MemoryLocation) error {
	var refs []patchRef
	for _, ml := range src {
		refs = append(refs, patchRef{PerformanceT, MemoryT, ml.index()})
	}
	destref := patchRef{PerformanceT, MemoryT, dest.index()}
	return memory.transfer(refs, destref, moveM)
}

func (memory *PatchMemory) MovePrograms(mls []MemoryLocation, dest MemoryLocation) error {
	var refs []patchRef
	for _, ml := range mls {
		refs = append(refs, patchRef{ProgramT, MemoryT, ml.index()})
	}
	destref := patchRef{ProgramT, MemoryT, dest.index()}
	return memory.transfer(refs, destref, moveM)
}

func (memory *PatchMemory) NumPerformances(onlyInitialized bool) int {
	result := NumPerformanceBanks * BankSize

	if onlyInitialized {
		for _, prog := range memory.performances {
			if prog == nil {
				result--
			}
		}
	}

	return result
}

func (memory *PatchMemory) NumPrograms(onlyInitialized bool) int {
	result := NumProgramBanks * BankSize

	if onlyInitialized {
		for _, prog := range memory.programs {
			if prog == nil {
				result--
			}
		}
	}

	return result
}

func (memory *PatchMemory) SprintPrograms(omitBlank bool) string {
	var result []string
	currBank := -1 // won't match any bank

	result = append(result, "\n***** PROGRAMS ******\n")

	for i := 0; i < len(memory.slotPrograms); i++ {
		result = append(result, fmt.Sprintf("Slot %d: %s", i+1, memory.slotPrograms[i].Summary()))
	}

	for i, program := range memory.programs {
		bank, location := bankloc(i)

		if bank != currBank {
			bank_header := fmt.Sprintf("\n*** Bank %v (%d/%d programs) ***", bank+1, memory.numInitialized(ProgramT, bank), BankSize)
			result = append(result, bank_header)
			currBank = bank
		}

		if memory.initialized(patchRef{ProgramT, MemoryT, i}) || !omitBlank {
			result = append(result, fmt.Sprintf("   %3d : %s", location+1, program.Summary()))
		}
	}

	return strings.Join(result, "\n")
}

func (memory *PatchMemory) SprintPerformances(omitBlank bool) string {
	var result []string
	currBank := -1 // won't match any bank

	result = append(result, "\n***** PERFORMANCES ******\n")
	result = append(result, fmt.Sprintf("Slot: %s", memory.slotPerformance.Summary()))

	for i, perf := range memory.performances {
		bank, location := bankloc(i)

		if bank != currBank {
			bank_header := fmt.Sprintf("\n*** Bank %v (%d/%d performances) ***", bank+1, memory.numInitialized(PerformanceT, bank), BankSize)
			result = append(result, bank_header)
			currBank = bank
		}

		if memory.initialized(patchRef{PerformanceT, MemoryT, i}) || !omitBlank {
			result = append(result, fmt.Sprintf("   %3d : %s", location+1, perf.Summary()))
		}
	}

	return strings.Join(result, "\n")
}

func (memory *PatchMemory) SwapPerformances(a MemoryLocation, b MemoryLocation) error {
	aref := patchRef{PerformanceT, MemoryT, a.index()}
	bref := patchRef{PerformanceT, MemoryT, a.index()}
	return memory.swap(aref, bref)
}

func (memory *PatchMemory) SwapPrograms(a MemoryLocation, b MemoryLocation) error {
	aref := patchRef{ProgramT, MemoryT, a.index()}
	bref := patchRef{ProgramT, MemoryT, a.index()}
	return memory.swap(aref, bref)
}

// Core internal behaviours

func (memory *PatchMemory) clear(ref patchRef) {
	if memory.initialized(ref) {
		switch ref.patchType {
		case PerformanceT:
			*memory.perfPtr(ref) = nil
		case ProgramT:
			*memory.progPtr(ref) = nil
		}
	}
}

func (memory *PatchMemory) copy(src patchRef, dest patchRef) error {
	if src.patchType != dest.patchType {
		return ErrXferTypeMismatch
	}
	if !src.valid() || !dest.valid() {
		return ErrInvalidLocation
	}
	if !memory.initialized(src) {
		return ErrUninitialized
	}
	if memory.initialized(dest) && dest.source != SlotT {
		return ErrMemoryOccupied // allow overwriting slots silently, they're temporary
	}

	switch src.patchType {
	case PerformanceT:
		srcPtr := memory.perfPtr(src)
		destPtr := memory.perfPtr(dest)
		copy := **srcPtr
		*destPtr = &copy
	case ProgramT:
		srcPtr := memory.progPtr(src)
		destPtr := memory.progPtr(dest)
		copy := **srcPtr
		*destPtr = &copy
	}
	return nil
}

// Formats a patch as sysex in NL3 format
func (memory *PatchMemory) export(ref patchRef) (*[]byte, error) {
	patch, err := memory.get(ref)
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

// Accepts an array of patchLocations and writes out fully formatted sysex blocks
func (memory *PatchMemory) exportLocations(refs []patchRef, writer io.Writer) error {
	var exportdata []byte

	for _, ref := range refs {
		fdata, err := memory.export(patchRef{ref.patchType, MemoryT, ref.index})
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
	_, err := writer.Write(exportdata)
	return err
}

// Returns a generic patch interface, could be either a program or a performance.
// It remains to the receiver to assert the patch to the necessary type.
func (memory *PatchMemory) get(ref patchRef) (patch, error) {
	var result patch

	if !ref.valid() {
		return nil, ErrInvalidLocation
	}
	if !memory.initialized(ref) {
		return nil, ErrUninitialized
	}
	switch ref.patchType {
	case PerformanceT:
		result = *memory.perfPtr(ref)
	case ProgramT:
		result = *memory.progPtr(ref)
	}

	return result, nil
}

func (memory *PatchMemory) importTo(s *sysex, ref patchRef, overwrite bool) error {
	if ref.patchType != s.patchType() {
		return ErrImportTypeMismatch
	}

	switch s.patchType() {
	case ProgramT:
		return memory.importProgram(s, ref, overwrite)
	case PerformanceT:
		return memory.importPerformance(s, ref, overwrite)
	}

	return nil
}

// Force sets the location in ref to the patch pointer, cast appropriately.
// Does not care if the location is already occupied (current contents will be lost if not previously copied to another location)
// Returns an error if the patch and ref are not the same type.
func (memory *PatchMemory) set(ref patchRef, patch patch) error {
	err := ErrInvalidLocation

	switch ref.patchType {
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

func (memory *PatchMemory) swap(src patchRef, dest patchRef) error {
	if src.patchType != dest.patchType {
		return ErrXferTypeMismatch
	}
	if src.source != dest.source {
		return ErrXferTypeMismatch // Don't support swapping to/from a slot, should be a copy or move.
	}
	if !src.valid() || !dest.valid() {
		return ErrInvalidLocation
	}

	switch src.patchType {
	case PerformanceT:
		srcPtr := memory.perfPtr(src)
		destPtr := memory.perfPtr(dest)
		temp := *destPtr
		*destPtr = *srcPtr
		*srcPtr = temp
	case ProgramT:
		srcPtr := memory.progPtr(src)
		destPtr := memory.progPtr(dest)
		temp := *destPtr
		*destPtr = *srcPtr
		*srcPtr = temp
	}
	return nil
}

// transfer can behave as a copy (mode is copyM) or a move (mode is moveM).
// Returns an error if any of the len(src) locations following dest are not empty, or if src contains
// patchLocations of different patchTypes
func (memory *PatchMemory) transfer(src []patchRef, dest patchRef, mode transferMode) error {
	var err error
	var moved []patchRef

	if len(src) == 0 {
		return nil
	}
	if src[0].patchType != dest.patchType {
		return ErrXferTypeMismatch
	}

	for i, currSrc := range src {
		currDest := patchRef{dest.patchType, dest.source, dest.index + i}

		if !currDest.valid() {
			err = ErrMemoryOverflow
		}
		if currSrc.patchType != currDest.patchType {
			err = ErrXferTypeMismatch
		}
		if memory.initialized(currDest) {
			err = ErrMemoryOccupied
		}
		if err != nil {
			memory.transfer(moved, src[0], moveM) // undo the ones moved so far
			break
		}

		// Handle move of each program type since their pointer values are separate
		switch currSrc.patchType {
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

// helpers

func (memory *PatchMemory) importPerformance(s *sysex, dest patchRef, overwrite bool) error {
	performanceData, err := newPerformanceFromBitstream(s.decodedBitstream)
	if err == nil {
		performance := Performance{
			name:     s.nameAsArray(),
			category: s.category(),
			version:  s.version(),
			data:     performanceData,
		}

		if existing, err := memory.get(dest); err == nil {
			if overwrite {
				fmt.Printf("Overwriting %s (%q) with %q\n", dest.String(), existing.PrintableName(), s.printableName())
			} else {
				return ErrMemoryOccupied
			}
		}
		err = memory.set(dest, &performance)
	}
	return err
}

func (memory *PatchMemory) importProgram(s *sysex, dest patchRef, overwrite bool) error {
	programData, err := newProgramFromBitstream(s.decodedBitstream)
	if err == nil {
		program := Program{
			name:     s.nameAsArray(),
			category: s.category(),
			version:  s.version(),
			data:     programData,
		}
		if existing, err := memory.get(dest); err == nil {
			if overwrite {
				fmt.Printf("Overwriting %s (%q) with %q\n", dest.String(), existing.PrintableName(), s.printableName())
			} else {
				return ErrMemoryOccupied
			}
		}
		err = memory.set(dest, &program)
	}
	return err
}

func (memory *PatchMemory) initialized(ref patchRef) (result bool) {
	if !ref.valid() {
		return
	}

	switch ref.source {
	case SlotT:
		switch ref.patchType {
		case PerformanceT:
			result = memory.slotPerformance != nil
		case ProgramT:
			result = memory.slotPrograms[ref.index] != nil
		}
	case MemoryT:
		switch ref.patchType {
		case PerformanceT:
			result = memory.performances[ref.index] != nil
		case ProgramT:
			result = memory.programs[ref.index] != nil
		}
	}

	return
}

func (memory *PatchMemory) numInitialized(pt PatchType, bank int) int {
	var result int
	offset := bank * BankSize

	for i := 0; i < BankSize; i++ {
		if memory.initialized(patchRef{pt, MemoryT, offset + i}) {
			result++
		}
	}

	return result
}

// panics if given an invalid patchRef
func (memory *PatchMemory) perfPtr(ref patchRef) (perf **Performance) {
	if ref.patchType != PerformanceT || !ref.valid() {
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
	if ref.patchType != ProgramT || !ref.valid() {
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

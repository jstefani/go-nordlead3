package nordlead3

import (
	"fmt"
	"testing"
)

func TestImportPerformance(t *testing.T) {
	memory := new(PatchMemory)
	sysex := validPerformanceSysex(t)
	helperLoadFromSysex(t, memory, sysex)

	performance, err := memory.get(validPerformanceRef)

	if err != nil {
		t.Errorf("Did not load performance into expected location!")
		return
	}

	if performance.PrintableName() != validPerformanceName {
		t.Errorf("Did not correctly compute the performance name.")
	}

	if performance.Version() != validPerformanceVersion {
		t.Errorf("Did not correctly compute the performance version.")
	}
}

func TestImportInvalidPerformance(t *testing.T) {
	memory := new(PatchMemory)
	sysex := invalidPerformanceSysex(t)
	helperLoadFromSysex(t, memory, sysex)

	_, err := memory.get(invalidPerformanceRef)

	if err == nil {
		t.Errorf("Loaded invalid performance into memory!")
	}
}

func TestImportProgram(t *testing.T) {
	memory := new(PatchMemory)
	sysex := validProgramSysex(t)
	helperLoadFromSysex(t, memory, sysex)

	patch, err := memory.get(validProgramRef)

	if err != nil {
		t.Errorf("Did not load program into expected location!")
		return
	}

	if patch.PrintableName() != validProgramName {
		t.Errorf("Did not correctly compute the program name: Expected %q, Got %q", validProgramName, patch.PrintableName())
	}

	if patch.Version() != validProgramVersion {
		t.Errorf("Did not correctly compute the program version.")
	}
}

func TestExportPerformance(t *testing.T) {
	memory := new(PatchMemory)
	inputSysex := validPerformanceSysex(t)
	helperLoadFromSysex(t, memory, inputSysex)

	outputSysex, err := memory.export(validPerformanceRef)
	if err != nil {
		t.Errorf("Error dumping program: %q", err)
	}

	binaryExpectEqual(t, &inputSysex, outputSysex)

	// test negative case
	uninitRef := patchRef{PerformanceT, MemoryT, index(validPerformanceBank, validPerformanceLocation+1)}
	outputSysex, err = memory.export(uninitRef)
	if err != ErrUninitialized {
		t.Errorf("Invalid error, expected ErrUninitialized, got %q", err)
	}
}

func TestExportProgram(t *testing.T) {
	memory := new(PatchMemory)
	inputSysex := validProgramSysex(t)
	helperLoadFromSysex(t, memory, inputSysex)

	outputSysex, err := memory.export(validProgramRef)
	if err != nil {
		t.Errorf("Error dumping program: %q", err)
	}

	binaryExpectEqual(t, &inputSysex, outputSysex)

	// test negative case
	outputSysex, err = memory.export(patchRef{ProgramT, MemoryT, index(validProgramBank+1, validProgramLocation+1)})
	if err != ErrUninitialized {
		t.Errorf("Invalid error, expected ErrUninitialized, got %q", err)
	}
}

func TestMovePrograms(t *testing.T) {
	memory := populatedMemory(t, "ProgBank1.syx")
	startLoc := 42
	numToMove := 27

	src := buildRefList(t, memory, ProgramT, 0, startLoc, numToMove, false)

	// Test successful moving to bank 2, same startLoc
	dest := patchRef{ProgramT, MemoryT, index(1, startLoc)}
	expectSuccessfulTransfer(t, memory, src, dest, moveM)

	// Test unsuccessful moving to end of range, expect overflow error
	src = buildRefList(t, memory, ProgramT, 1, startLoc, numToMove, false)
	oneOverMax := (NumProgramBanks+1)*BankSize - numToMove + 1
	dest = patchRef{ProgramT, MemoryT, oneOverMax}
	expectUnsuccessfulTransfer(t, memory, src, dest, ErrMemoryOverflow, moveM)

	// Test unsuccessful moving to occupied range
	dest = patchRef{ProgramT, MemoryT, index(0, 127)}
	expectUnsuccessfulTransfer(t, memory, src, dest, ErrMemoryOccupied, moveM)
}

func TestMovePerformances(t *testing.T) {
	memory := populatedMemory(t, "PerfBank1.syx")
	startBank := 0
	startLoc := 42
	numToMove := 27
	destBank := 1

	// Test successful moving to bank 2, same startLoc
	src := buildRefList(t, memory, PerformanceT, startBank, startLoc, numToMove, false)
	dest := patchRef{PerformanceT, MemoryT, index(destBank, startLoc)}
	expectSuccessfulTransfer(t, memory, src, dest, moveM)

	// Test unsuccessful moving to end of range, expect overflow error
	src = buildRefList(t, memory, PerformanceT, destBank, startLoc, numToMove, false)
	oneOverMax := (NumPerformanceBanks+1)*BankSize - numToMove + 1
	dest = patchRef{PerformanceT, MemoryT, oneOverMax}
	expectUnsuccessfulTransfer(t, memory, src, dest, ErrMemoryOverflow, moveM)

	// Test unsuccessful moving to occupied range
	dest = patchRef{PerformanceT, MemoryT, index(destBank, startLoc+numToMove-1)}
	expectUnsuccessfulTransfer(t, memory, src, dest, ErrMemoryOccupied, moveM)
}

func TestSwapPerformances(t *testing.T) {
	memory := populatedMemory(t, "PerfBank1.syx")

	// A. Test swap of two occupied regions
	a := patchRef{PerformanceT, MemoryT, index(0, 42)}
	b := patchRef{PerformanceT, MemoryT, index(0, 43)}
	expectSuccessfulSwap(t, memory, a, b)

	// B. Test swap of an occupied region to an unoccupied region
	a = patchRef{PerformanceT, MemoryT, index(0, 42)}
	b = patchRef{PerformanceT, MemoryT, index(1, 42)}
	requireUninitialized(t, memory, b)
	expectSuccessfulSwap(t, memory, a, b)

	// C. Test the converse of B, now that they are reversed
	a = patchRef{PerformanceT, MemoryT, index(0, 42)}
	b = patchRef{PerformanceT, MemoryT, index(1, 42)}
	requireUninitialized(t, memory, a)
	expectSuccessfulSwap(t, memory, a, b)

	// D. Test error handling when a swap is requested between the wrong types
	a = patchRef{PerformanceT, MemoryT, index(0, 42)}
	b = patchRef{ProgramT, MemoryT, index(1, 42)}
	expectUnsuccessfulSwap(t, memory, a, b, ErrXferTypeMismatch)

	// E. Test error handling when a swap is requested from an invalid location
	a = patchRef{PerformanceT, MemoryT, index(NumPerformanceBanks, 42)}
	b = patchRef{PerformanceT, MemoryT, index(1, 42)}
	expectUnsuccessfulSwap(t, memory, a, b, ErrInvalidLocation)

	// F. Test error handling whan a swap is requested to an invalid location
	a = patchRef{PerformanceT, MemoryT, index(1, 42)}
	b = patchRef{PerformanceT, MemoryT, index(NumPerformanceBanks, 42)}
	expectUnsuccessfulSwap(t, memory, a, b, ErrInvalidLocation)
}

func TestSwapPrograms(t *testing.T) {
	memory := populatedMemory(t, "ProgBank1.syx")

	// A. Test swap of two occupied regions
	a := patchRef{ProgramT, MemoryT, index(0, 42)}
	b := patchRef{ProgramT, MemoryT, index(0, 43)}
	expectSuccessfulSwap(t, memory, a, b)

	// B. Test swap of an occupied region to an unoccupied region
	a = patchRef{ProgramT, MemoryT, index(0, 42)}
	b = patchRef{ProgramT, MemoryT, index(1, 42)}
	requireUninitialized(t, memory, b)
	expectSuccessfulSwap(t, memory, a, b)

	// C. Test the converse of B, now that they are reversed
	a = patchRef{ProgramT, MemoryT, index(0, 42)}
	b = patchRef{ProgramT, MemoryT, index(1, 42)}
	requireUninitialized(t, memory, a)
	expectSuccessfulSwap(t, memory, a, b)

	// E. Test error handling when a swap is requested from an invalid location
	a = patchRef{ProgramT, MemoryT, index(NumProgramBanks, 42)}
	b = patchRef{ProgramT, MemoryT, index(1, 42)}
	expectUnsuccessfulSwap(t, memory, a, b, ErrInvalidLocation)

	// F. Test error handling whan a swap is requested to an invalid location
	a = patchRef{ProgramT, MemoryT, index(1, 42)}
	b = patchRef{ProgramT, MemoryT, index(NumProgramBanks, 42)}
	expectUnsuccessfulSwap(t, memory, a, b, ErrInvalidLocation)
}

func TestCopyPerformanceToSlot(t *testing.T) {
	memory := populatedMemory(t, "PerfBank1.syx")

	// A. Copy a valid performance to the slot
	ml := MemoryLocation{0, 42}
	err := memory.CopyPerformanceToSlot(ml)
	src := patchRef{PerformanceT, MemoryT, ml.index()}
	dest := performanceSlotRef
	expectSuccessfulCopy(t, memory, src, dest, err)

	// B. Copy a performance to the slot when the slot already has a performance
	ml = MemoryLocation{0, 43}
	err = memory.CopyPerformanceToSlot(ml)
	src = patchRef{PerformanceT, MemoryT, ml.index()}
	dest = performanceSlotRef
	expectSuccessfulCopy(t, memory, src, dest, err)

	// C. Copy an uninitialized performance to the slot
	ml = MemoryLocation{1, 42}
	err = memory.CopyPerformanceToSlot(ml)
	src = patchRef{PerformanceT, MemoryT, ml.index()}
	dest = performanceSlotRef
	expectUnsuccessfulCopy(t, memory, src, dest, err, ErrUninitialized)

	// TODO: Invalid and other edge cases not tested at present.
}

func TestCopyPerformanceFromSlot(t *testing.T) {
	memory := populatedMemory(t, "PerfSlot.syx")

	// A. Copy a valid performance from the slot
	ml := MemoryLocation{0, 42}
	err := memory.CopySlotToPerformance(ml)
	src := performanceSlotRef
	dest := patchRef{PerformanceT, MemoryT, ml.index()}
	expectSuccessfulCopy(t, memory, src, dest, err)

	// B. Copy from slot when slot is uninitialized
	memory.slotPerformance = nil
	ml = MemoryLocation{1, 42}
	err = memory.CopySlotToPerformance(ml)
	src = performanceSlotRef
	dest = patchRef{PerformanceT, MemoryT, ml.index()}
	expectUnsuccessfulCopy(t, memory, src, dest, err, ErrUninitialized)

	// TODO: Invalid and other edge cases not tested at present.
}

func TestCopyProgramToSlot(t *testing.T) {
	memory := populatedMemory(t, "ProgBank1.syx")
	targetSlot := 1

	// A. Copy a valid program to the slot
	ml := MemoryLocation{0, 42}
	err := memory.CopyProgramToSlot(ml, targetSlot)
	src := patchRef{ProgramT, MemoryT, ml.index()}
	dest := patchRef{ProgramT, SlotT, targetSlot}
	expectSuccessfulCopy(t, memory, src, dest, err)

	// B. Copy a program to the slot when the slot already has a program
	ml = MemoryLocation{0, 43}
	src = patchRef{ProgramT, MemoryT, ml.index()}
	dest = patchRef{ProgramT, SlotT, targetSlot}
	err = memory.CopyProgramToSlot(ml, targetSlot)
	expectSuccessfulCopy(t, memory, src, dest, err)

	// C. Copy an uninitialized program to the slot, expect failure
	ml = MemoryLocation{1, 42}
	err = memory.CopyProgramToSlot(ml, targetSlot)
	expectUnsuccessfulCopy(t, memory, src, dest, err, ErrUninitialized)
}

func TestCopyProgramFromSlot(t *testing.T) {
	memory := populatedMemory(t, "ProgBank1.syx")
	targetSlot := 1

	// Setup: stick a program in the slot, we test this elsewhere, so assume it works
	ml := MemoryLocation{0, 42}
	memory.CopyProgramToSlot(ml, targetSlot)

	// A. Copy a valid program from the slot to a blank destination
	ml = MemoryLocation{1, 42}
	err := memory.CopySlotToProgram(targetSlot, ml)
	src := patchRef{ProgramT, SlotT, targetSlot}
	dest := patchRef{ProgramT, MemoryT, ml.index()}
	expectSuccessfulCopy(t, memory, src, dest, err)

	// B. Copy a valid program from the slot to an initialized destination, expect error
	ml = MemoryLocation{0, 43}
	err = memory.CopySlotToProgram(targetSlot, ml)
	src = patchRef{ProgramT, SlotT, targetSlot}
	dest = patchRef{ProgramT, MemoryT, ml.index()}
	expectUnsuccessfulCopy(t, memory, src, dest, err, ErrMemoryOccupied)

	// C. Copy a program from a slot that is not initialized, expect error
	ml = MemoryLocation{1, 42}
	targetSlot = 0
	err = memory.CopySlotToProgram(targetSlot, ml)
	src = patchRef{ProgramT, SlotT, targetSlot}
	dest = patchRef{ProgramT, MemoryT, ml.index()}
	expectUnsuccessfulCopy(t, memory, src, dest, err, ErrUninitialized)
}

func TestDeletePerformance(t *testing.T) {
	memory := populatedMemory(t, "PerfBank1.syx")
	targetLocation := MemoryLocation{0, 42}
	targetRef := patchRef{PerformanceT, MemoryT, targetLocation.index()}
	requireInitialized(t, memory, targetRef)
	memory.DeletePerformance(targetLocation)
	if p, err := memory.get(targetRef); err != ErrUninitialized {
		t.Errorf("Attempted to delete %s but retrieved %s after deletion", targetRef.String(), p.Summary())
	}
}

func TestDeleteProgram(t *testing.T) {
	memory := populatedMemory(t, "ProgBank1.syx")
	targetLocation := MemoryLocation{0, 42}
	targetRef := patchRef{ProgramT, MemoryT, targetLocation.index()}
	requireInitialized(t, memory, targetRef)
	memory.DeleteProgram(targetLocation)
	if p, err := memory.get(targetRef); err != ErrUninitialized {
		t.Errorf("Attempted to delete %s but retrieved %s after deletion", targetRef.String(), p.Summary())
	}
}

// Helpers =======================================================

func buildRefList(t *testing.T, memory *PatchMemory, pt PatchType, startBank, startLocation, numToMove int, permitBlank bool) (refs []patchRef) {
	startLoc := index(startBank, startLocation)

	for i := startLoc; i < startLoc+numToMove; i++ {
		ref := patchRef{pt, MemoryT, i}
		_, err := memory.get(ref)

		if !permitBlank && err != nil {
			continue // ignore uninitialized patches in move block
		}
		refs = append(refs, ref)
	}
	if !permitBlank && len(refs) < 10 {
		t.Fatalf("Test range %s %d:%d-%d:%d does not contain a sufficient quantity of initialized patches.", pt.String(), startBank, startLocation, bank(index(startBank, startLocation)+numToMove), location(index(startBank, startLocation)+numToMove))
	}
	return
}

func expectSuccessfulCopy(t *testing.T, memory *PatchMemory, src, dest patchRef, err error) {
	if err != nil {
		t.Errorf("Got error copying performance to slot: %s", err)
	}
	from, _ := memory.get(src)
	to, _ := memory.get(dest)
	if to == nil {
		t.Errorf("Copy seems to have erased the original")
	}
	if from == to {
		t.Errorf("Duplicate pointers, did not copy value")
	}
	if from.Summary() != to.Summary() {
		t.Errorf("Copy failed: expected %s, got %s", from.Summary(), to.Summary())
	}
}

func expectSuccessfulSwap(t *testing.T, memory *PatchMemory, aref patchRef, bref patchRef) {
	a, aerr := memory.get(aref)
	b, berr := memory.get(bref)

	err := memory.swap(aref, bref)
	if err != nil {
		t.Errorf("Error swapping %s with %s: %s", aref.String(), bref.String(), err)
	}
	// Ensure the swap occurred
	a2, aerr2 := memory.get(aref)
	b2, berr2 := memory.get(bref)

	if a2 != b || aerr2 != berr {
		var aStr, a2Str string
		if aerr != nil {
			aStr = "uninitialized"
		} else {
			aStr = a.Summary()
		}
		if aerr2 != nil {
			a2Str = "uninitialized"
		} else {
			a2Str = a2.Summary()
		}
		t.Errorf("Swap incorrectly changed %s from %s to %s", aref.String(), aStr, a2Str)
	}
	if b2 != a || berr2 != aerr {
		var bStr, b2Str string
		if berr != nil {
			bStr = "uninitialized"
		} else {
			bStr = b.Summary()
		}
		if berr2 != nil {
			b2Str = "uninitialized"
		} else {
			b2Str = b2.Summary()
		}
		t.Errorf("Swap incorrectly changed %s from %s to %s", bref.String(), bStr, b2Str)
	}
}

func expectSuccessfulTransfer(t *testing.T, memory *PatchMemory, src []patchRef, dest patchRef, mode transferMode) {
	dests := buildRefList(t, memory, dest.patchType, dest.bank(), dest.location(), len(src), true)
	quicksummary := summarize(memory, src)

	err := memory.transfer(src, dest, mode)
	if err != nil {
		t.Fatalf("Failure moving patches: %s", err)
	}
	for i := 0; i < len(src); i++ {
		if mode == moveM {
			_, err = memory.get(src[i])
			if err != ErrUninitialized {
				t.Fatalf("Move left original location %d:%d initialized", src[i].bank(), src[i].location())
			}
		}

		moved, err := memory.get(dests[i])
		if err == ErrUninitialized {
			t.Fatalf("Transfer left destination location %d:%d uninitialized", dests[i].bank(), dests[i].location())
		}
		msum := fmt.Sprintf("%s:%f", moved.PrintableName(), moved.Version())
		if msum != quicksummary[i] {
			t.Fatalf("Did not correctly transfer patch: Expected destination to contain %q, got %q", quicksummary[i], msum)
		}
	}
}

func expectUnsuccessfulCopy(t *testing.T, memory *PatchMemory, src, dest patchRef, err error, expectedError error) {
	if err != expectedError {
		t.Errorf("Expected %q when copying uninitialized patch, but got %q", expectedError, err)
	}
}

func expectUnsuccessfulSwap(t *testing.T, memory *PatchMemory, aref patchRef, bref patchRef, expectedError error) {
	a, aerr := memory.get(aref)
	b, berr := memory.get(bref)

	err := memory.swap(aref, bref)
	if err != expectedError {
		t.Errorf("Expected error %s, got: %s", expectedError, err)
	}
	// Ensure the swap did not occur
	if a2, aerr2 := memory.get(aref); a2 != a || aerr2 != aerr {
		var aStr, a2Str string
		if aerr != nil {
			aStr = "uninitialized"
		} else {
			aStr = a.Summary()
		}
		if aerr2 != nil {
			a2Str = "uninitialized"
		} else {
			a2Str = a2.Summary()
		}
		t.Errorf("Failed swap changed %s from %s to %s", aref.String(), aStr, a2Str)
	}
	if b2, berr2 := memory.get(bref); b2 != b || berr2 != berr {
		var bStr, b2Str string
		if berr != nil {
			bStr = "uninitialized"
		} else {
			bStr = b.Summary()
		}
		if berr2 != nil {
			b2Str = "uninitialized"
		} else {
			b2Str = b2.Summary()
		}
		t.Errorf("Failed swap changed %s from %s to %s", bref.String(), bStr, b2Str)
	}
}

func expectUnsuccessfulTransfer(t *testing.T, memory *PatchMemory, src []patchRef, dest patchRef, expectedError error, mode transferMode) {
	quicksummary := summarize(memory, src)
	err := memory.transfer(src, dest, mode)
	if err != expectedError {
		t.Errorf("Expected error %s, got: %s", expectedError, err)
	}

	// Validate that it put everything back
	for i := 0; i < len(src); i++ {
		moved, err := memory.get(src[i])
		if err == ErrUninitialized {
			t.Fatalf("Failed move left original %d:%d uninitialized", 0, i)
		}
		msum := fmt.Sprintf("%s:%f", moved.PrintableName(), moved.Version())
		if msum != quicksummary[i] {
			t.Errorf("Did not correctly restore patch after failed move: Expected %q, got %q", quicksummary[i], msum)
		}

		if mode == copyM {
			// test that copy didn't leave clutter behind
			dests := buildRefList(t, memory, dest.patchType, dest.bank(), dest.location(), len(src), true)
			_, err = memory.get(dests[i])
			if err != ErrUninitialized {
				t.Fatalf("Failed copy left destination location %d:%d initialized when it should be blank", src[i].bank(), src[i].location())
			}
		}
	}
}

func requireInitialized(t *testing.T, memory *PatchMemory, ref patchRef) {
	if _, err := memory.get(ref); err != nil {
		t.Fatalf("Expected %s to be initialized, but got error %s", ref.String(), err)
	}
}

func requireUninitialized(t *testing.T, memory *PatchMemory, ref patchRef) {
	if p, err := memory.get(ref); err != ErrUninitialized {
		t.Fatalf("Expected %s to be uninitialized, but it contained %s", ref.String(), p.Summary())
	}
}

func summarize(memory *PatchMemory, refs []patchRef) (summaries []string) {
	for _, ref := range refs {
		if p, err := memory.get(ref); err == nil {
			summaries = append(summaries, fmt.Sprintf("%s:%f", p.PrintableName(), p.Version()))
		} else {
			summaries = append(summaries, "")
		}
	}
	return
}

package nordlead3

import (
	"fmt"
	"testing"
)

func TestLoadValidPerformanceFromSysex(t *testing.T) {
	memory := new(PatchMemory)
	sysex := ValidPerformanceSysex(t)

	err := memory.Import(sysex)
	if err != nil {
		t.Errorf("Expected clean load from valid sysex. Got: %q", err)
	}

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

func TestLoadInvalidPerformanceFromSysex(t *testing.T) {
	memory := new(PatchMemory)
	sysex := InvalidPerformanceSysex(t)

	err := memory.Import(sysex)
	if err == nil {
		t.Errorf("Expected error from invalid sysex")
		return
	}

	_, err = memory.get(invalidPerformanceRef)

	if err == nil {
		t.Errorf("Loaded invalid performance into memory!")
	}
}

func TestLoadProgramFromSysex(t *testing.T) {
	memory := new(PatchMemory)
	sysex := ValidProgramSysex(t)

	err := memory.Import(sysex)
	if err != nil {
		t.Errorf("Expected clean load from valid sysex. Got: %q", err)
	}

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

func TestDumpProgramToSysex(t *testing.T) {
	memory := new(PatchMemory)
	inputSysex := ValidProgramSysex(t)
	err := memory.Import(inputSysex)
	if err != nil {
		t.Fatalf("Test sysex seems incorrect, need valid sysex to test dumping: %q", err)
	}

	outputSysex, err := memory.export(validProgramRef)
	if err != nil {
		t.Errorf("Error dumping program: %q", err)
	}

	BinaryExpectEqual(t, &inputSysex, outputSysex)

	// test negative case
	outputSysex, err = memory.export(patchRef{programT, memoryT, index(validProgramBank+1, validProgramLocation+1)})
	if err != ErrUninitialized {
		t.Errorf("Invalid error, expected ErrUninitialized, got %q", err)
	}
}

func TestDumpPerformanceToSysex(t *testing.T) {
	memory := new(PatchMemory)
	inputSysex := ValidPerformanceSysex(t)
	err := memory.Import(inputSysex)
	if err != nil {
		t.Fatalf("Test sysex seems incorrect, need valid sysex to test dumping: %q", err)
	}

	outputSysex, err := memory.export(validPerformanceRef)
	if err != nil {
		t.Errorf("Error dumping program: %q", err)
	}

	BinaryExpectEqual(t, &inputSysex, outputSysex)

	// test negative case
	uninitRef := patchRef{performanceT, memoryT, index(validPerformanceBank, validPerformanceLocation+1)}
	outputSysex, err = memory.export(uninitRef)
	if err != ErrUninitialized {
		t.Errorf("Invalid error, expected ErrUninitialized, got %q", err)
	}
}

// TODO: This is a mess, clean it up!
func TestMovePrograms(t *testing.T) {
	var src []patchRef
	var quicksummary []string
	var dest []patchRef
	var err error
	// populate a PatchMemory
	memory := new(PatchMemory)
	startLoc := 42
	numToMove := 27

	helperLoadFromFile(t, memory, "ProgBank1.syx")

	src = buildRefList(t, memory, programT, 0, startLoc, numToMove, false)
	quicksummary = summarize(memory, src)

	// Test successful moving to bank 2, same startLoc
	dest = buildRefList(t, memory, programT, 1, startLoc, numToMove, true)
	err = memory.Transfer(src, dest[0], moveM)
	if err != nil {
		t.Fatalf("Failure moving patches: %s", err)
	}
	for i := 0; i < numToMove; i++ {
		_, err = memory.get(src[i])
		if err != ErrUninitialized {
			t.Fatalf("Move left original location %d:%d initialized", 0, i)
		}
		moved, _ := memory.get(dest[i])
		msum := fmt.Sprintf("%s:%f", moved.PrintableName(), moved.Version())
		if msum != quicksummary[i] {
			t.Fatalf("Did not correctly move patch: Expected %q, got %q", quicksummary[i], msum)
		}
	}

	// Test unsuccessful moving to end of range, expect overflow error
	src = buildRefList(t, memory, programT, 1, startLoc, numToMove, false)
	quicksummary = summarize(memory, src)
	oneOverMax := bankSize - numToMove + 1
	dest = buildRefList(t, memory, programT, numProgBanks-1, oneOverMax, numToMove, true)

	err = memory.Transfer(src, dest[0], moveM)
	if err != ErrMemoryOverflow {
		t.Errorf("Expected error ErrorMemoryOverflow, got: %s", err)
	}

	// Validate that it put everything back
	for i := 0; i < numToMove; i++ {
		orig, err := memory.get(src[i])
		if err == ErrUninitialized {
			t.Fatalf("Failed move left original %d:%d uninitialized", src[i].bank(), src[i].location())
		}
		msum := fmt.Sprintf("%s:%f", orig.PrintableName(), orig.Version())
		if msum != quicksummary[i] {
			t.Errorf("Did not correctly restore patch after failed move: Expected %q, got %q", quicksummary[i], msum)
		}
	}

	// Test unsuccessful moving to occupied range
	err = memory.Transfer(src, patchRef{programT, memoryT, index(0, 127)}, moveM)
	if err != ErrMemoryOccupied {
		t.Errorf("Expected error ErrorMemoryOccupied, got %s", err)
	}
}

// TODO: This is a mess, clean it up!
func TestMovePerformances(t *testing.T) {
	var src []patchRef
	var quicksummary []string
	var dest []patchRef
	var err error
	// populate a PatchMemory
	memory := new(PatchMemory)
	startBank := 0
	startLoc := 42
	numToMove := 27
	destBank := 1

	helperLoadFromFile(t, memory, "PerfBank1.syx")

	src = buildRefList(t, memory, performanceT, startBank, startLoc, numToMove, false)
	quicksummary = summarize(memory, src)

	// Test successful moving to bank 2, same startLoc
	dest = buildRefList(t, memory, performanceT, destBank, startLoc, numToMove, true)
	err = memory.Transfer(src, dest[0], moveM)
	if err != nil {
		t.Fatalf("Failure moving patches: %s", err)
	}
	for i := 0; i < numToMove; i++ {
		_, err = memory.get(src[i])
		if err != ErrUninitialized {
			t.Fatalf("Move left original location %d:%d initialized", startBank, i)
		}
		moved, _ := memory.get(dest[i])
		msum := fmt.Sprintf("%s:%f", moved.PrintableName(), moved.Version())
		if msum != quicksummary[i] {
			t.Fatalf("Did not correctly move patch: Expected %q, got %q", quicksummary[i], msum)
		}
	}

	// Test unsuccessful moving to end of range, expect overflow error
	src = buildRefList(t, memory, performanceT, destBank, startLoc, numToMove, false)
	quicksummary = summarize(memory, src)
	oneOverMax := (numPerfBanks*bankSize - 1) - (numToMove - 2)
	err = memory.Transfer(src, patchRef{performanceT, memoryT, oneOverMax}, moveM)
	if err != ErrMemoryOverflow {
		t.Errorf("Expected error ErrorMemoryOverflow, got: %s", err)
	}

	// Validate that it put everything back
	for i := 0; i < numToMove; i++ {
		moved, err := memory.get(src[i])
		if err == ErrUninitialized {
			t.Fatalf("Failed move left original %d:%d uninitialized", 0, i)
		}
		msum := fmt.Sprintf("%s:%f", moved.PrintableName(), moved.Version())
		if msum != quicksummary[i] {
			t.Errorf("Did not correctly restore patch after failed move: Expected %q, got %q", quicksummary[i], msum)
		}
	}

	// Test unsuccessful moving to occupied range
	err = memory.Transfer(src, patchRef{performanceT, memoryT, index(destBank, startLoc+numToMove-1)}, moveM)
	if err != ErrMemoryOccupied {
		t.Errorf("Expected error ErrorMemoryOccupied, got %s", err)
	}
}

// TODO: This is a mess, clean it up!
func buildRefList(t *testing.T, memory *PatchMemory, pt patchType, startBank, startLocation, numToMove int, permitBlank bool) (refs []patchRef) {
	startLoc := index(startBank, startLocation)

	for i := startLoc; i < startLoc+numToMove; i++ {
		ref := patchRef{pt, memoryT, i}
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

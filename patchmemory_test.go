package nordlead3

import (
	"fmt"
	"testing"
)

func TestLoadValidPerformanceFromSysex(t *testing.T) {
	memory := new(PatchMemory)
	sysex := ValidPerformanceSysex(t)

	err := memory.LoadFromSysex(sysex)
	if err != nil {
		t.Errorf("Expected clean load from valid sysex. Got: %q", err)
	}

	performance, err := memory.GetPerformance(validPerformanceBank, validPerformanceLocation)

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

	err := memory.LoadFromSysex(sysex)
	if err == nil {
		t.Errorf("Expected error from invalid sysex")
		return
	}

	_, err = memory.GetPerformance(invalidPerformanceBank, invalidPerformanceLocation)

	if err == nil {
		t.Errorf("Loaded invalid performance into memory!")
	}
}

func TestLoadProgramFromSysex(t *testing.T) {
	memory := new(PatchMemory)
	sysex := ValidProgramSysex(t)

	err := memory.LoadFromSysex(sysex)
	if err != nil {
		t.Errorf("Expected clean load from valid sysex. Got: %q", err)
	}

	program, err := memory.GetProgram(validProgramBank, validProgramLocation)

	if err != nil {
		t.Errorf("Did not load program into expected location!")
		return
	}

	if program.PrintableName() != validProgramName {
		t.Errorf("Did not correctly compute the program name: Expected %q, Got %q", validProgramName, program.PrintableName())
	}

	if program.Version() != validProgramVersion {
		t.Errorf("Did not correctly compute the program version.")
	}
}

func TestDumpProgramToSysex(t *testing.T) {
	memory := new(PatchMemory)
	inputSysex := ValidProgramSysex(t)
	err := memory.LoadFromSysex(inputSysex)
	if err != nil {
		t.Fatalf("Test sysex seems incorrect, need valid sysex to test dumping: %q", err)
	}

	outputSysex, err := memory.DumpProgram(validProgramBank, validProgramLocation)
	if err != nil {
		t.Errorf("Error dumping program: %q", err)
	}

	BinaryExpectEqual(t, &inputSysex, outputSysex)

	// test negative case
	outputSysex, err = memory.DumpProgram(validProgramBank+1, validProgramLocation+1)
	if err != ErrorUninitialized {
		t.Errorf("Invalid error, expected ErrorUninitialized, got %q", err)
	}
}

func TestDumpPerformanceToSysex(t *testing.T) {
	memory := new(PatchMemory)
	inputSysex := ValidPerformanceSysex(t)
	err := memory.LoadFromSysex(inputSysex)
	if err != nil {
		t.Fatalf("Test sysex seems incorrect, need valid sysex to test dumping: %q", err)
	}

	outputSysex, err := memory.DumpPerformance(validPerformanceBank, validPerformanceLocation)
	if err != nil {
		t.Errorf("Error dumping program: %q", err)
	}

	BinaryExpectEqual(t, &inputSysex, outputSysex)

	// test negative case
	outputSysex, err = memory.DumpPerformance(validPerformanceBank+1, validPerformanceLocation+1)
	if err != ErrorUninitialized {
		t.Errorf("Invalid error, expected ErrorUninitialized, got %q", err)
	}
}

func TestMovePrograms(t *testing.T) {
	var src []patchRef
	var quicksummary []string
	var dest patchRef
	var err error
	// populate a PatchMemory
	memory := new(PatchMemory)
	startLoc := 42
	numToMove := 27

	helperLoadFromFile(t, memory, "ProgBank1.syx")

	src, quicksummary = buildSrcList(t, memory, programT, 0, startLoc, numToMove)

	// Test successful moving to bank 2, same startLoc
	dest = patchRef{programT, index(1, startLoc)}
	err = memory.move(src, dest)
	if err != nil {
		t.Fatalf("Failure moving patches: %s", err)
	}
	for i := startLoc; i < startLoc+numToMove; i++ {
		_, err = memory.GetProgram(0, i)
		if err != ErrorUninitialized {
			t.Fatalf("Move left original location %d:%d initialized", 0, i)
		}
		moved, _ := memory.GetProgram(1, i)
		msum := fmt.Sprintf("%s:%s:%f", moved.PrintableName(), moved.PrintableCategory(), moved.Version())
		if msum != quicksummary[i-startLoc] {
			t.Fatalf("Did not correctly move patch: Expected %q, got %q", quicksummary[i], msum)
		}
	}

	// Test unsuccessful moving to end of range, expect overflow error
	src, quicksummary = buildSrcList(t, memory, programT, 1, startLoc, numToMove)
	oneOverMax := (numProgBanks*bankSize - 1) - (numToMove - 2)
	dest = patchRef{programT, oneOverMax}
	err = memory.move(src, dest)
	if err != ErrorMemoryOverflow {
		t.Errorf("Expected error ErrorMemoryOverflow, got: %s", err)
	}

	// Validate that it put everything back
	for i := startLoc; i < startLoc+numToMove; i++ {
		moved, err := memory.GetProgram(1, i)
		if err == ErrorUninitialized {
			t.Fatalf("Failed move left original %d:%d uninitialized", 0, i)
		}
		msum := fmt.Sprintf("%s:%s:%f", moved.PrintableName(), moved.PrintableCategory(), moved.Version())
		if msum != quicksummary[i-startLoc] {
			t.Errorf("Did not correctly restore patch after failed move: Expected %q, got %q", quicksummary[i], msum)
		}
	}

	// Test unsuccessful moving to occupied range
	dest = patchRef{programT, index(0, 127)}
	err = memory.move(src, dest)
	if err != ErrorMemoryOccupied {
		t.Errorf("Expected error ErrorMemoryOccupied, got %s", err)
	}
}

func TestMovePerformances(t *testing.T) {
	var src []patchRef
	var quicksummary []string
	var dest patchRef
	var err error
	// populate a PatchMemory
	memory := new(PatchMemory)
	startBank := 0
	startLoc := 42
	numToMove := 27
	destBank := 1

	helperLoadFromFile(t, memory, "PerfBank1.syx")

	src, quicksummary = buildSrcList(t, memory, performanceT, startBank, startLoc, numToMove)

	// Test successful moving to bank 2, same startLoc
	dest = patchRef{programT, index(destBank, startLoc)}
	err = memory.move(src, dest)
	if err != nil {
		t.Fatalf("Failure moving patches: %s", err)
	}
	for i := startLoc; i < startLoc+numToMove; i++ {
		_, err = memory.GetPerformance(startBank, i)
		if err != ErrorUninitialized {
			t.Fatalf("Move left original location %d:%d initialized", startBank, i)
		}
		moved, _ := memory.GetPerformance(destBank, i)
		msum := fmt.Sprintf("%s:%f", moved.PrintableName(), moved.Version())
		if msum != quicksummary[i-startLoc] {
			t.Fatalf("Did not correctly move patch: Expected %q, got %q", quicksummary[i], msum)
		}
	}

	// Test unsuccessful moving to end of range, expect overflow error
	src, quicksummary = buildSrcList(t, memory, performanceT, destBank, startLoc, numToMove)
	oneOverMax := (numPerfBanks*bankSize - 1) - (numToMove - 2)
	dest = patchRef{programT, oneOverMax}
	err = memory.move(src, dest)
	if err != ErrorMemoryOverflow {
		t.Errorf("Expected error ErrorMemoryOverflow, got: %s", err)
	}

	// Validate that it put everything back
	for i := startLoc; i < startLoc+numToMove; i++ {
		moved, err := memory.GetPerformance(1, i)
		if err == ErrorUninitialized {
			t.Fatalf("Failed move left original %d:%d uninitialized", 0, i)
		}
		msum := fmt.Sprintf("%s:%f", moved.PrintableName(), moved.Version())
		if msum != quicksummary[i-startLoc] {
			t.Errorf("Did not correctly restore patch after failed move: Expected %q, got %q", quicksummary[i], msum)
		}
	}

	// Test unsuccessful moving to occupied range
	dest = patchRef{performanceT, index(destBank, startLoc+numToMove-1)}
	err = memory.move(src, dest)
	if err != ErrorMemoryOccupied {
		t.Errorf("Expected error ErrorMemoryOccupied, got %s", err)
	}
}

func buildSrcList(t *testing.T, memory *PatchMemory, pt patchType, startBank, startLocation, numToMove int) (refs []patchRef, summaries []string) {
	var err error
	var oprog *Program
	var operf *Performance

	startLoc := index(startBank, startLocation)
	for i := startLoc; i < startLoc+numToMove; i++ {
		if pt == programT {
			oprog, err = memory.GetProgram(0, i)
			if err != nil {
				summaries = append(summaries, "")
				continue // ignore uninitialized patches in move block
			}
			refs = append(refs, patchRef{programT, i})
			summaries = append(summaries, fmt.Sprintf("%s:%s:%f", oprog.PrintableName(), oprog.PrintableCategory(), oprog.Version()))
		} else {
			operf, err = memory.GetPerformance(0, i)
			if err != nil {
				summaries = append(summaries, "")
				continue // ignore uninitialized patches in move block
			}
			refs = append(refs, patchRef{performanceT, i})
			summaries = append(summaries, fmt.Sprintf("%s:%f", operf.PrintableName(), operf.Version()))
		}
	}
	if len(refs) < 10 {
		t.Fatalf("Test range %d:%d-%d:%d does not contain a sufficient quantity of initialized patches.", startBank, startLocation, bank(index(startBank, startLocation)+numToMove), location(index(startBank, startLocation)+numToMove))
	}
	return refs, summaries
}

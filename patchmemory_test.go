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
		t.Errorf("Test sysex seems incorrect, need valid sysex to test dumping: %q", err)
		return
	}

	outputSysex, err := memory.DumpProgram(validProgramBank, validProgramLocation)
	if err != nil {
		t.Errorf("Error dumping program: %q", err)
		return
	}

	BinaryExpectEqual(t, &inputSysex, outputSysex)
}

func TestDumpPerformanceToSysex(t *testing.T) {
	memory := new(PatchMemory)
	inputSysex := ValidPerformanceSysex(t)
	err := memory.LoadFromSysex(inputSysex)
	if err != nil {
		t.Errorf("Test sysex seems incorrect, need valid sysex to test dumping: %q", err)
		return
	}

	outputSysex, err := memory.DumpPerformance(validPerformanceBank, validPerformanceLocation)
	if err != nil {
		t.Errorf("Error dumping program: %q", err)
		return
	}

	BinaryExpectEqual(t, &inputSysex, outputSysex)
}

func TestMovePrograms(t *testing.T) {
	var src []patchLocation
	var quicksummary []string
	var dest patchLocation
	var err error
	// populate a PatchMemory
	memory := new(PatchMemory)
	startPos := 42
	numToMove := 27

	helperLoadFromFile(t, memory, "ProgBank1.syx")

	for i := startPos; i < startPos+numToMove; i++ {
		orig, err := memory.GetProgram(0, uint8(i))
		if err != nil {
			quicksummary = append(quicksummary, "")
			continue // ignore uninitialized patches in move block
		}
		src = append(src, patchLocation{patchProgram, 0, uint8(i)})
		quicksummary = append(quicksummary, fmt.Sprintf("%s:%s:%f", orig.PrintableName(), orig.PrintableCategory(), orig.Version()))
	}
	if len(src) < 10 {
		t.Errorf("Test range does not contain a sufficient quantity of initialized patches.")
		return
	}

	// Test successful moving to bank 2, same startPos
	dest = patchLocation{patchProgram, 1, uint8(startPos)}
	err = memory.move(src, dest)
	if err != nil {
		t.Fatalf("Failure moving patches: %s", err)
	}
	for i := startPos; i < startPos+numToMove; i++ {
		_, err = memory.GetProgram(0, uint8(i))
		if err != ErrorUninitialized {
			t.Fatalf("Move left original location %d:%d initialized", 0, i)
		}
		moved, _ := memory.GetProgram(1, uint8(i))
		msum := fmt.Sprintf("%s:%s:%f", moved.PrintableName(), moved.PrintableCategory(), moved.Version())
		if msum != quicksummary[i-startPos] {
			t.Fatalf("Did not correctly move patch: Expected %q, got %q", quicksummary[i], msum)
		}
	}
}

func TestMovePerformances(t *testing.T) {
	var src []patchLocation
	var quicksummary []string
	var dest patchLocation
	var err error
	// populate a PatchMemory
	memory := new(PatchMemory)
	startPos := 42
	numToMove := 27

	helperLoadFromFile(t, memory, "PerfBank1.syx")

	for i := startPos; i < startPos+numToMove; i++ {
		orig, err := memory.GetPerformance(0, uint8(i))
		if err != nil {
			quicksummary = append(quicksummary, "")
			continue // ignore uninitialized patches in move block
		}
		src = append(src, patchLocation{patchPerformance, 0, uint8(i)})
		quicksummary = append(quicksummary, fmt.Sprintf("%s:%f", orig.PrintableName(), orig.Version()))
	}
	if len(src) < 10 {
		t.Errorf("Test range does not contain a sufficient quantity of initialized patches.")
		return
	}

	// Test successful moving to bank 2, same startPos
	dest = patchLocation{patchProgram, 1, uint8(startPos)}
	err = memory.move(src, dest)
	if err != nil {
		t.Fatalf("Failure moving patches: %s", err)
	}
	for i := startPos; i < startPos+numToMove; i++ {
		_, err = memory.GetProgram(0, uint8(i))
		if err != ErrorUninitialized {
			t.Fatalf("Move left original location %d:%d initialized", 0, i)
		}
		moved, _ := memory.GetPerformance(1, uint8(i))
		msum := fmt.Sprintf("%s:%f", moved.PrintableName(), moved.Version())
		if msum != quicksummary[i-startPos] {
			t.Fatalf("Did not correctly move patch: Expected %q, got %q", quicksummary[i], msum)
		}
	}
}

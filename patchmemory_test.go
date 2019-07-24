package nordlead3

import (
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
	}

	_, err = memory.GetPerformance(validPerformanceBank, validPerformanceLocation)

	if err != nil {
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
		t.Errorf("Did not correctly compute the program name.")
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

package nordlead3

import (
	"testing"
)

func TestDumpPerformanceSysex(t *testing.T) {
	memory := new(PatchMemory)
	inputSysex := ValidPerformanceSysex(t)
	inputSysexStruct, err := ParseSysex(inputSysex)
	if err != nil {
		t.Errorf("Test sysex seems incorrect, need valid sysex to test dumping: %q", err)
	}
	performanceSysex := inputSysexStruct.rawBitstream()

	err = memory.LoadFromSysex(inputSysex)
	if err != nil {
		t.Errorf("Test sysex seems incorrect, need valid sysex to test dumping: %q", err)
	}
	performance := memory.performances[validPerformanceBank][validPerformanceLocation].performance

	outputSysex, err := performance.dumpSysex()
	if err != nil {
		t.Errorf("Error dumping performance: %q", err)
	}

	// Compare the decoded data for easier debugging
	decodedPS := unpackSysex(performanceSysex)
	decodedOS := unpackSysex(*outputSysex)
	location, explanation := LocationOfDifference(&decodedPS, &decodedOS)
	if explanation != nil {
		t.Errorf("Dumped sysex does not match input at offset %d (%d): %q", location, location*8, explanation)
	}
}

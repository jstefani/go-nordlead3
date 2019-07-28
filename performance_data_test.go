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

	err = memory.Import(inputSysex)
	if err != nil {
		t.Errorf("Test sysex seems incorrect, need valid sysex to test dumping: %q", err)
	}

	p, err := memory.get(validPerformanceRef)
	performance := p.(*Performance)
	if err != nil {
		t.Errorf("Error retrieving performance: %q", err)
	}

	outputSysex, err := performance.data.dumpSysex()
	if err != nil {
		t.Errorf("Error dumping performance: %q", err)
	}

	// Compare the decoded data for easier debugging
	decodedPS := unpackSysex(performanceSysex)
	decodedOS := unpackSysex(*outputSysex)
	BinaryExpectEqual(t, &decodedPS, &decodedOS)
}

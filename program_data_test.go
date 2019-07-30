package nordlead3

import (
	"testing"
)

func TestDumpProgramSysex(t *testing.T) {
	memory := new(PatchMemory)
	inputSysex := validProgramSysex(t)
	inputSysexStruct, err := parseSysex(inputSysex)
	if err != nil {
		t.Errorf("Test sysex seems incorrect, need valid sysex to test dumping: %q", err)
	}
	programSysex := inputSysexStruct.rawBitstream()

	helperLoadFromSysex(t, memory, inputSysex)
	p, err := memory.get(validProgramRef)
	program := p.(*Program)

	outputSysex, err := program.data.dumpSysex()
	if err != nil {
		t.Errorf("Error dumping program: %q", err)
	}

	// Compare the decoded data for easier debugging
	decodedPS := unpackSysex(programSysex)
	decodedOS := unpackSysex(*outputSysex)
	location, explanation := locationOfDifference(&decodedPS, &decodedOS)
	if explanation != nil {
		t.Errorf("Dumped sysex does not match input at offset %d (%d): %q", location, location*8, explanation)
	}
}

func TestPackAndUnpackSysex(t *testing.T) {
	s, _ := parseSysex(validProgramSysex(t))
	bitsToRepack := s.decodedBitstream
	repackedBits := unpackSysex(packSysex(bitsToRepack))

	if string(bitsToRepack) != string(repackedBits) {
		t.Errorf("Pack and Unpack not symmetric: %x / %x", tailBytes(bitsToRepack, 8), tailBytes(repackedBits, 8))
	}
}

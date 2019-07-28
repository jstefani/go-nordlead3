package nordlead3

import (
	"testing"
)

func TestDumpProgramSysex(t *testing.T) {
	memory := new(PatchMemory)
	inputSysex := ValidProgramSysex(t)
	inputSysexStruct, err := ParseSysex(inputSysex)
	if err != nil {
		t.Errorf("Test sysex seems incorrect, need valid sysex to test dumping: %q", err)
	}
	programSysex := inputSysexStruct.rawBitstream()

	err = memory.Import(inputSysex)
	if err != nil {
		t.Errorf("Test sysex seems incorrect, need valid sysex to test dumping: %q", err)
	}
	p, err := memory.Get(validProgramRef)
	program := p.(*Program)

	outputSysex, err := program.data.dumpSysex()
	if err != nil {
		t.Errorf("Error dumping program: %q", err)
	}

	// Compare the decoded data for easier debugging
	decodedPS := unpackSysex(programSysex)
	decodedOS := unpackSysex(*outputSysex)
	location, explanation := LocationOfDifference(&decodedPS, &decodedOS)
	if explanation != nil {
		t.Errorf("Dumped sysex does not match input at offset %d (%d): %q", location, location*8, explanation)
	}
}

func TestPackAndUnpackSysex(t *testing.T) {
	sysex, _ := ParseSysex(ValidProgramSysex(t))
	bitsToRepack := sysex.decodedBitstream
	repackedBits := unpackSysex(packSysex(bitsToRepack))

	if string(bitsToRepack) != string(repackedBits) {
		t.Errorf("Pack and Unpack not symmetric: %x / %x", tailBytes(bitsToRepack, 8), tailBytes(repackedBits, 8))
	}
}

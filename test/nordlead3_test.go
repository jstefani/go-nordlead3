package nordlead3_test

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/malacalypse/nordlead3"
	"io"
	"io/ioutil"
	"path/filepath"
	"testing"
)

const (
	validPerformanceBank     = 0
	validPerformanceLocation = 0
	validPerformanceName     = "Orchestra     HN"
	validPerformanceVersion  = 1.20
	validProgramBank         = 2
	validProgramLocation     = 2
	validProgramName         = "BladeRun     ZON"
	validProgramVersion      = 1.18
)

func TestLoadValidPerformanceFromSysex(t *testing.T) {
	memory := new(nordlead3.PatchMemory)
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
	memory := new(nordlead3.PatchMemory)
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
	memory := new(nordlead3.PatchMemory)
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
	memory := new(nordlead3.PatchMemory)
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
	memory := new(nordlead3.PatchMemory)
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

// If error is not nil, int holds location and error holds a regional comparison for debugging.
// If error is nil, there was no difference.
func LocationOfDifference(pb1, pb2 *[]byte) (int, error) {
	b1 := *pb1
	b2 := *pb2
	r1 := bytes.NewReader(b1)
	r2 := bytes.NewReader(b2)
	i := 0

	for {
		c1, err1 := r1.ReadByte()
		c2, err2 := r2.ReadByte()
		if c1 == c2 && err1 == err2 {
			if err1 == io.EOF {
				return 0, nil
			}
		} else {
			minIndex := Max(0, i-5)
			maxIndex1 := Min(i+5, len(b1))
			maxIndex2 := Min(i+5, len(b2))
			explanation := fmt.Sprintf("Bytes 1: %x^%x | Bytes 2: %x^%x", b1[minIndex:i], b1[i:maxIndex1], b2[minIndex:i], b2[i:maxIndex2])
			return i, errors.New(explanation)
		}
		i++
	}
}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func ValidPerformanceSysex(t *testing.T) []byte {
	return helperLoadBytes(t, "Performance-Orchestra     HN.syx")
}
func ValidProgramSysex(t *testing.T) []byte {
	return helperLoadBytes(t, "Program-BladeRun     ZON-1.18.syx")
}

func InvalidPerformanceSysex(t *testing.T) []byte {
	return helperLoadBytes(t, "Performance-Invalid.syx")
}

func InvalidProgramSysex(t *testing.T) []byte {
	return helperLoadBytes(t, "Program-Invalid.syx")
}

func BinaryExpectEqual(t *testing.T, expected *[]byte, received *[]byte) {
	location, explanation := LocationOfDifference(expected, received)
	if explanation != nil {
		fmt.Printf("Expected:  %x\n", expected)
		fmt.Printf("Received: %x\n", received)
		t.Errorf("Dumped sysex does not match input at offset %d: %q", location, explanation)
	}
}

func helperLoadBytes(t *testing.T, name string) []byte {
	path := filepath.Join("../testdata", name) // relative path
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return bytes
}

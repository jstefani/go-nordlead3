package nordlead3

// Test Utilities

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

const (
	validPerformanceBank       = 1
	validPerformanceLocation   = 2
	invalidPerformanceBank     = 0
	invalidPerformanceLocation = 1
	validPerformanceName       = "Orchestra     HN"
	validPerformanceVersion    = 1.20
	validProgramBank           = 3
	validProgramLocation       = 4
	validProgramName           = "Blade run    ZON"
	validProgramVersion        = 1.18
)

var validPerformanceRef = patchRef{PerformanceT, MemoryT, index(validPerformanceBank, validPerformanceLocation)}
var invalidPerformanceRef = patchRef{PerformanceT, MemoryT, index(invalidPerformanceBank, invalidPerformanceLocation)}
var validProgramRef = patchRef{ProgramT, MemoryT, index(validProgramBank, validProgramLocation)}

// If error is not nil, int holds location and error holds a regional comparison for debugging.
// If error is nil, there was no difference.
func locationOfDifference(pb1, pb2 *[]byte) (int, error) {
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
			minIndex := max(0, i-5)
			maxIndex1 := min(i+5, len(b1))
			maxIndex2 := min(i+5, len(b2))
			explanation := fmt.Sprintf("Bytes 1: %x^%x | Bytes 2: %x^%x", b1[minIndex:i], b1[i:maxIndex1], b2[minIndex:i], b2[i:maxIndex2])
			return i, errors.New(explanation)
		}
		i++
	}
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func validPerformanceSysex(t *testing.T) []byte {
	return helperLoadBytes(t, "Performance-Orchestra     HN.syx")
}
func validProgramSysex(t *testing.T) []byte {
	return helperLoadBytes(t, "Program-BladeRun     ZON-1.18.syx")
}

func invalidPerformanceSysex(t *testing.T) []byte {
	return helperLoadBytes(t, "Performance-Invalid.syx")
}

func invalidProgramSysex(t *testing.T) []byte {
	return helperLoadBytes(t, "Program-Invalid.syx")
}

func binaryExpectEqual(t *testing.T, expected *[]byte, received *[]byte) {
	location, explanation := locationOfDifference(expected, received)
	if explanation != nil {
		fmt.Printf("Expected: %x\n", expected)
		fmt.Printf("Received: %x\n", received)
		t.Errorf("Dumped sysex does not match input at offset %d: %q", location, explanation)
	}
}

func helperLoadBytes(t *testing.T, name string) []byte {
	path := filepath.Join("testdata", name) // relative path
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return bytes
}

func helperLoadFromFile(t *testing.T, memory *PatchMemory, filename string) {
	file, err := os.Open(filepath.Join("testdata", filename))
	defer file.Close()

	if err != nil {
		t.Fatal(fmt.Printf("Could not open %q: %q\n", filename, err))
	}
	memory.ImportFrom(file)
}

func tailBytes(buf []byte, n int) []byte {
	start := max(0, len(buf)-n)
	return buf[start:]
}

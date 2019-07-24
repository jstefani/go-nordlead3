package nordlead3

import (
	"fmt"
	"strings"
)

type ProgramLocation struct {
	name     [16]byte
	category uint8
	version  float64
	program  *Program
}

func (progLoc *ProgramLocation) PrintableName() string {
	if progLoc == nil {
		return "** Uninitialized"
	}
	return fmt.Sprintf("%-16s", strings.TrimRight(string(progLoc.name[:]), "\x00"))
}

func (progLoc *ProgramLocation) Summary() string {
	if progLoc == nil {
		return "** Uninitialized"
	}
	return fmt.Sprintf("%+-16.16q (%1.2f)", progLoc.PrintableName(), progLoc.version)
}

func (progLoc *ProgramLocation) Version() float64 {
	return progLoc.version
}

func (progLoc *ProgramLocation) PrintContents(depth int) {
	if progLoc == nil {
		fmt.Println(strUninitializedName)
	}
	fmt.Printf("Printing %16q (%1.2f) {\n", progLoc.PrintableName(), progLoc.version)

	printStruct(progLoc.program, depth)
}

package nordlead3

import (
	"fmt"
	"strings"
)

type ProgramLocation struct {
	Name     [16]byte
	Category uint8
	Version  float64
	Program  *Program
}

func (progLoc *ProgramLocation) PrintableName() string {
	if progLoc == nil {
		return "** Uninitialized"
	}
	return fmt.Sprintf("%-16s", strings.TrimRight(string(progLoc.Name[:]), "\x00"))
}

func (progLoc *ProgramLocation) summary() string {
	if progLoc == nil {
		return "** Uninitialized"
	}
	return fmt.Sprintf("%+-16.16q (%1.2f)", progLoc.PrintableName(), progLoc.Version)
}

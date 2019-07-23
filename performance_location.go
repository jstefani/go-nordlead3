package nordlead3

import (
	"fmt"
	"strings"
)

type PerformanceLocation struct {
	Name        [16]byte
	Category    uint8
	Version     float64
	Performance *Performance
}

func (perfLoc *PerformanceLocation) PrintableName() string {
	if perfLoc == nil {
		return "** Uninitialized"
	}
	return fmt.Sprintf("%-16s", strings.TrimRight(string(perfLoc.Name[:]), "\x00"))
}

func (perfLoc *PerformanceLocation) summary() string {
	if perfLoc == nil {
		return "** Uninitialized"
	}
	return fmt.Sprintf("%16.16q (%1.2f)", perfLoc.PrintableName(), perfLoc.Version)
}

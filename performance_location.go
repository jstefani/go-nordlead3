package nordlead3

import (
	"fmt"
	"strings"
)

type PerformanceLocation struct {
	name        [16]byte
	category    uint8
	version     float64
	performance *Performance
}

func (perfLoc *PerformanceLocation) PrintableName() string {
	if perfLoc == nil {
		return strUninitializedName
	}
	return fmt.Sprintf("%-16s", strings.TrimRight(string(perfLoc.name[:]), "\x00"))
}

func (perfLoc *PerformanceLocation) Summary() string {
	if perfLoc == nil {
		return strUninitializedName
	}
	return fmt.Sprintf("%16.16q (%1.2f)", perfLoc.PrintableName(), perfLoc.version)
}

func (perfLoc *PerformanceLocation) Version() float64 {
	return perfLoc.version
}

func (perfLoc *PerformanceLocation) PrintContents(depth int) {
	if perfLoc == nil {
		fmt.Println(strUninitializedName)
	}
	fmt.Printf("Printing %16q (%1.2f)\n", perfLoc.PrintableName(), perfLoc.version)
	printStruct(perfLoc.performance, depth)
}

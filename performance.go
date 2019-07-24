package nordlead3

import (
	"fmt"
	"strings"
)

type Performance struct {
	name     [16]byte
	category uint8
	version  float64
	data     *PerformanceData
}

func (performance *Performance) PrintableName() string {
	if performance == nil {
		return strUninitializedName
	}
	return fmt.Sprintf("%-16s", strings.TrimRight(string(performance.name[:]), "\x00"))
}

func (performance *Performance) Summary() string {
	if performance == nil {
		return strUninitializedName
	}
	return fmt.Sprintf("%16.16q (%1.2f)", performance.PrintableName(), performance.version)
}

func (performance *Performance) Version() float64 {
	return performance.version
}

func (performance *Performance) PrintContents(depth int) {
	if performance == nil {
		fmt.Println(strUninitializedName)
	}
	fmt.Printf("Printing %16q (%1.2f)\n", performance.PrintableName(), performance.version)
	printStruct(performance.data, depth)
}

// Implement sysexable

func (performance *Performance) sysexCategory() uint8 {
	return performance.category
}

func (performance *Performance) sysexData() (*[]byte, error) {
	return performance.data.dumpSysex()
}

func (performance *Performance) sysexName() []byte {
	result := make([]byte, 16)

	for i := 0; i < 16; i++ {
		currByte := performance.name[i]
		if uint8(currByte) < 128 {
			result[i] = currByte
		} else {
			result[i] = 0x2D // "-"
		}
	}

	return result
}

func (performance *Performance) sysexType() uint8 {
	return PerformanceFromMemory
}

func (performance *Performance) sysexVersion() []byte {
	versionX100 := uint16(performance.version * 100)
	return []byte{byte(versionX100 >> 8), byte(versionX100)}
}

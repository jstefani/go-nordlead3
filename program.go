package nordlead3

import (
	"fmt"
	"strings"
)

type Program struct {
	name     [16]byte
	category uint8
	version  float64
	data     *ProgramData
}

// Implement patch

func (program *Program) PatchType() PatchType {
	return ProgramT
}

func (program *Program) PrintContents(depth int) {
	if program == nil {
		fmt.Println(strUninitializedName)
	}
	fmt.Printf("Printing %16q (%1.2f) {\n", program.PrintableName(), program.version)

	printStruct(program.data, depth)
}

func (program *Program) PrintableCategory() string {
	if program == nil {
		return ""
	}
	return Categories[program.category]
}

func (program *Program) PrintableName() string {
	if program == nil {
		return strUninitializedName
	}
	return fmt.Sprintf("%-16s", strings.TrimRight(string(program.name[:]), "\x00"))
}

func (program *Program) SetCategory(newCategory uint8) error {
	if program == nil {
		return ErrUninitialized // can't set a category on an uninitialized program
	}
	if newCategory > 0x0D {
		return ErrInvalidCategory
	}
	program.category = newCategory
	return nil
}

func (program *Program) SetName(newName string) error {
	if program == nil {
		return ErrUninitialized // can't set a category on an uninitialized program
	}

	var byteName [16]byte

	if len(newName) > 16 || len(newName) == 0 {
		return ErrInvalidName
	}
	copy(byteName[:], newName)
	program.name = byteName
	return nil
}

func (program *Program) Summary() string {
	if program == nil {
		return strUninitializedName
	}
	return fmt.Sprintf("%+-16.16q : %8s (%1.2f)", program.PrintableName(), program.PrintableCategory(), program.version)
}

func (program *Program) Version() float64 {
	return program.version
}

// Implement sysexable

func (program *Program) sysexCategory() uint8 {
	return program.category
}

func (program *Program) sysexData() (*[]byte, error) {
	return program.data.dumpSysex()
}

func (program *Program) sysexName() []byte {
	result := make([]byte, 16)

	for i := 0; i < 16; i++ {
		currByte := program.name[i]
		if uint8(currByte) < 128 {
			result[i] = currByte
		} else {
			result[i] = 0x2D // "-"
		}
	}

	return result
}

func (program *Program) sysexType() uint8 {
	return ProgramFromMemory
}

func (program *Program) sysexVersion() []byte {
	versionX100 := uint16(program.version * 100)
	return []byte{byte(versionX100 >> 8), byte(versionX100)}
}

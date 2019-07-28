package nordlead3

import (
	"fmt"
)

// patchTypes
const (
	ProgramT = iota
	PerformanceT
)

// sourceTypes
const (
	SlotT = iota
	MemoryT
)

type PatchType int
type SourceType int

func (pt PatchType) String() (typeStr string) {
	switch PatchType(pt) {
	case PerformanceT:
		typeStr = "performance"
	case ProgramT:
		typeStr = "program"
	}
	return
}

type patchRef struct {
	PatchType PatchType
	source    SourceType
	index     int
}

func (ref *patchRef) bank() int {
	return bank(ref.index)
}

func (ref *patchRef) location() int {
	return location(ref.index)
}

func (ref *patchRef) valid() bool {
	return valid(ref.PatchType, ref.source, ref.index)
}

func (ref *patchRef) String() string {
	var sourceStr string
	var typeStr string
	var locationStr string

	switch ref.PatchType {
	case PerformanceT:
		typeStr = "performance"
	case ProgramT:
		typeStr = "program"
	}
	switch ref.source {
	case SlotT:
		sourceStr = "slot"
		locationStr = fmt.Sprintf("%d", ref.index)
	case MemoryT:
		sourceStr = "memory location"
		locationStr = fmt.Sprintf("%d:%d", bank(ref.index), location(ref.index))
	}

	return fmt.Sprintf("%s in %s %s", typeStr, sourceStr, locationStr)
}

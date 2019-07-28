package nordlead3

import (
	"fmt"
)

// patchTypes
const (
	programT = iota
	performanceT
)

// sourceTypes
const (
	slotT = iota
	memoryT
)

type patchType int
type sourceType int

func (pt patchType) String() (typeStr string) {
	switch patchType(pt) {
	case performanceT:
		typeStr = "performance"
	case programT:
		typeStr = "program"
	}
	return
}

type patchRef struct {
	patchType patchType
	source    sourceType
	index     int
}

func (ref *patchRef) bank() int {
	return bank(ref.index)
}

func (ref *patchRef) location() int {
	return location(ref.index)
}

func (ref *patchRef) valid() bool {
	return valid(ref.patchType, ref.source, ref.index)
}

func (ref *patchRef) String() string {
	var sourceStr string
	var typeStr string
	var locationStr string

	switch ref.patchType {
	case performanceT:
		typeStr = "performance"
	case programT:
		typeStr = "program"
	}
	switch ref.source {
	case slotT:
		sourceStr = "slot"
		locationStr = fmt.Sprintf("%d", ref.index)
	case memoryT:
		sourceStr = "memory location"
		locationStr = fmt.Sprintf("%d:%d", bank(ref.index), location(ref.index))
	}

	return fmt.Sprintf("%s in %s %s", typeStr, sourceStr, locationStr)
}

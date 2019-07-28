package nordlead3

func bank(index int) int {
	return index / bankSize
}

// func bankv(pt PatchType, index int) (int, bool) {
// 	return index / bankSize, valid(pt, index)
// }

func location(index int) int {
	return index % bankSize
}

// func locationv(pt PatchType, index int) (int, bool) {
// 	return index % bankSize, valid(pt, index)
// }

func index(bank, location int) int {
	return bank*bankSize + location
}

// func indexv(pt PatchType, bank, location int) (int, bool) {
// 	index := bank*bankSize + location
// 	return index, valid(pt, index)
// }

// Useful when we know the location is valid already
func bankloc(index int) (int, int) {
	return bank(index), location(index)
}

func valid(pt PatchType, st SourceType, index int) (result bool) {
	var numBanks int

	switch st {
	case SlotT:
		switch pt {
		case PerformanceT:
			result = index == 0
		case ProgramT:
			result = index >= 0 && index < 4
		}
	case MemoryT:
		switch pt {
		case PerformanceT:
			numBanks = numPerfBanks
		case ProgramT:
			numBanks = numProgBanks
		}
		result = index >= 0 && index < numBanks*bankSize
	}
	return
}

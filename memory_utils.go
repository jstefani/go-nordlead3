package nordlead3

func bank(index int) int {
	return index / bankSize
}

// func bankv(pt patchType, index int) (int, bool) {
// 	return index / bankSize, valid(pt, index)
// }

func location(index int) int {
	return index % bankSize
}

// func locationv(pt patchType, index int) (int, bool) {
// 	return index % bankSize, valid(pt, index)
// }

func index(bank, location int) int {
	return bank*bankSize + location
}

// func indexv(pt patchType, bank, location int) (int, bool) {
// 	index := bank*bankSize + location
// 	return index, valid(pt, index)
// }

// Useful when we know the location is valid already
func bankloc(index int) (int, int) {
	return bank(index), location(index)
}

func valid(pt patchType, st sourceType, index int) (result bool) {
	var numBanks int

	switch st {
	case slotT:
		switch pt {
		case performanceT:
			result = index == 0
		case programT:
			result = index >= 0 && index < 4
		}
	case memoryT:
		switch pt {
		case performanceT:
			numBanks = numPerfBanks
		case programT:
			numBanks = numProgBanks
		}
		result = index >= 0 && index < numBanks*bankSize
	}
	return
}

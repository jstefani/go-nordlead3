package nordlead3

// A convenience to consolidate function arguments
type MemoryLocation struct {
	Bank     int
	Location int
}

func (ml MemoryLocation) index() int {
	return index(ml.Bank, ml.Location)
}

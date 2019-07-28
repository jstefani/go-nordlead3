package nordlead3

// A convenience to consolidate function arguments
type MemoryLocation struct {
	bank     int
	location int
}

func (ml MemoryLocation) index() int {
	return index(ml.bank, ml.location)
}

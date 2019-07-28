package nordlead3

type patch interface {
	SetCategory(uint8) error
	SetName(string) error
	PrintContents(int)
	PrintableCategory() string
	PrintableName() string
	Summary() string
	Version() float64
	PatchType() PatchType
}

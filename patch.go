package nordlead3

type patch interface {
	SetCategory(int) error
	SetName(string) error
	PrintContents(int)
	PrintableCategory() string
	PrintableName() string
	Summary() string
	Version() float64
	PatchType() PatchType
}

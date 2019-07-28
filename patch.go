package nordlead3

type patch interface {
	SetName(string) error
	PrintableContents(int)
	PrintableName() string
	Summary() string
	Version() float64
	patchType() patchType
}

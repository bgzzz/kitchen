package shelves

// Shelf is a structure defining the kitchen shelf options
type Shelf struct {
	Name               string
	Temp               string
	Capacity           int
	ShelfDecayModifier int
}

const (
	// OverflowShelfTemp defines temperature label of the overflow shell
	OverflowShelfTemp = "any"
)

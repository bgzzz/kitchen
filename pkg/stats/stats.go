package stats

import "fmt"

// Stats structure to keep common stats
type Stats struct {
	wastedValues    []float64
	deliveredValues []float64
	spoiled         int
	expected        int
}

// NewStats creates new stats object
func NewStats(expected int) *Stats {
	return &Stats{
		expected: expected,
	}
}

// Delivered add delivered values to the stats
func (st *Stats) Delivered(v float64) {
	st.deliveredValues = append(st.deliveredValues, v)
}

// Wasted add wasted values to the stats
func (st *Stats) Wasted(v float64) {
	st.wastedValues = append(st.wastedValues, v)
}

// Spoiled add spoiled cases to the stats
func (st *Stats) Spoiled() {
	st.spoiled++
}

// AvgWasted return average value of wasted orders
func (st *Stats) AvgWasted() float64 {
	if len(st.wastedValues) == 0 {
		return 0
	}

	var sum float64
	for _, v := range st.wastedValues {
		sum += v
	}

	return sum / float64(len(st.wastedValues))
}

// AvgDelivered return average value of delivered orders
func (st *Stats) AvgDelivered() float64 {
	if len(st.deliveredValues) == 0 {
		return 0
	}

	var sum float64
	for _, v := range st.deliveredValues {
		sum += v
	}

	return sum / float64(len(st.deliveredValues))
}

// String return formatted output of the gathered stats
func (st *Stats) String() string {
	return fmt.Sprintf("\n\tDelivered %d/%d, avg value %f\n"+
		"\tWasted %d/%d, avg value %f\n"+
		"\tSpoiled %d/%d", len(st.deliveredValues), st.expected,
		st.AvgDelivered(),
		len(st.wastedValues), st.expected, st.AvgWasted(),
		st.spoiled, st.expected)
}

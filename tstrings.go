package vnix

import "strconv"

// TStrings is a slice of strings with numeric conversion helpers.
type TStrings []string

// Strings returns the values as a plain string slice.
func (this TStrings) Strings() []string {
	return []string(this)
}

// Ints converts each value to an integer, using def when conversion fails.
func (this TStrings) Ints(def int) []int {
	r := make([]int, len(this))

	for i, v := range this {
		n, err := strconv.Atoi(v)

		if err != nil {
			n = def
		}
		r[i] = n
	}
	return r
}

// Float32s converts each value to float32, using def when conversion fails.
func (this TStrings) Float32s(def float32) []float32 {
	r := make([]float32, len(this))

	for i, v := range this {
		n, err := strconv.ParseFloat(v, 32)

		if err != nil {
			r[i] = def
			continue
		}
		r[i] = float32(n)
	}
	return r
}

// Float64s converts each value to float64, using def when conversion fails.
func (this TStrings) Float64s(def float64) []float64 {
	r := make([]float64, len(this))

	for i, v := range this {
		n, err := strconv.ParseFloat(v, 64)

		if err != nil {
			n = def
		}
		r[i] = n
	}
	return r
}

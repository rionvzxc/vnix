package vnix

import "strconv"

// TString is a string value with numeric conversion helpers.
type TString string

// String returns the string value.
func (this TString) String() string {
	return string(this)
}

// Int converts the value to an integer or returns def if conversion fails.
func (this TString) Int(def int) int {
	v, err := strconv.Atoi(this.String())
	if err != nil {
		return def
	}
	return v
}

// Float32 converts the value to a float32 or returns def if conversion fails.
func (this TString) Float32(def float32) float32 {
	v, err := strconv.ParseFloat(this.String(), 32)
	if err != nil {
		return def
	}
	return float32(v)
}

// Float64 converts the value to a float64 or returns def if conversion fails.
func (this TString) Float64(def float64) float64 {
	v, err := strconv.ParseFloat(this.String(), 64)
	if err != nil {
		return def
	}
	return v
}

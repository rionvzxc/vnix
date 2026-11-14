package vnix

import "strconv"

type TString string

func (this TString) String() string {
	return string(this)
}

func (this TString) Int(def int) int {
	v, err := strconv.Atoi(this.String())
	if err != nil {
		return def
	}
	return v
}

func (this TString) Float32(def float32) float32 {
	v, err := strconv.ParseFloat(this.String(), 32)
	if err != nil {
		return def
	}
	return float32(v)
}

func (this TString) Float64(def float64) float64 {
	v, err := strconv.ParseFloat(this.String(), 64)
	if err != nil {
		return def
	}
	return v
}

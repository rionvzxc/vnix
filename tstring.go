package vnix

import (
	"strconv"
)

type TString string

func (this TString) Int(def int) int {
	v, err := strconv.Atoi(string(this))
	if err != nil { return def }
	return v
}
func (this TString) Float(def float64) float64 {
	v, err := strconv.ParseFloat(string(this), 64)
	if err != nil { return def }
	return v
}
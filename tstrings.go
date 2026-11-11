package vnix

import (
	"strconv"
)

type TStrings []string

func (this TStrings) Str(def string) string {
	if len(this) == 0 { return def }
	return this[0]
}
func (this TStrings) Int(def int) int {
	v, err := strconv.Atoi(this.Str(""))
	if err != nil { return def }
	return v
}
func (this TStrings) Float(def float64) float64 {
	v, err := strconv.ParseFloat(this.Str(""), 64)
	if err != nil { return def }
	return v
}
func (this TStrings) StrA() []string {
	return []string(this)
}
func (this TStrings) IntA(def int) []int {
	r := make([]int, len(this))
	for i, v := range this {
		n, err := strconv.Atoi(v)
		if err != nil { n = def }
		r[i] = n
	}
	return r
}
func (this TStrings) FloatA(def float64) []float64 {
	r := make([]float64, len(this))
	for i, v := range this {
		n, err := strconv.ParseFloat(v, 64)
		if err != nil { n = def }
		r[i] = n
	}
	return r
}
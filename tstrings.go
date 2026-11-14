package vnix

import "strconv"

type TStrings []string

func (this TStrings) String(def string) string {
	if len(this) == 0 {
		return def
	}
	return this[0]
}

func (this TStrings) Int(def int) int {
	v, err := strconv.Atoi(this.String(""))
	if err != nil {
		return def
	}
	return v
}

func (this TStrings) Float32(def float32) float32 {
	v, err := strconv.ParseFloat(this.String(""), 32)
	if err != nil {
		return def
	}
	return float32(v)
}

func (this TStrings) Float64(def float64) float64 {
	v, err := strconv.ParseFloat(this.String(""), 64)
	if err != nil {
		return def
	}
	return v
}

func (this TStrings) Strings() []string {
	return []string(this)
}

func (this TStrings) IntA(def int) []int {
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

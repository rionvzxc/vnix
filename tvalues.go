package vnix

type TValues map[string][]string

func (this *TValues) Set(data TValues) {
	*this = data
}
func (this TValues) Add(name string, values ...string) {
	this[name] = append(this[name], values...)
}
func (this TValues) Del(name string) {
	delete(this, name)
}
func (this TValues) Exist(name string) bool {
	_, exists := this[name]
	return exists
}
func (this TValues) Get(name string) string {
	if values := this[name]; len(values) != 0 {
		return values[0]
	}
	return ""
}
func (this TValues) GetA(name string) []string {
	return this[name]
}
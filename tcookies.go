package vnix

// TCookies is a collection of HTTP cookies.
type TCookies []*TCookie

// Set replaces the collection with data.
func (this *TCookies) Set(data TCookies) {
	*this = data
}
// Add appends copies of the supplied cookies to the collection.
func (this *TCookies) Add(cookies ...TCookie) {
	for _, cookie := range cookies {
		cookieCopy := cookie
		*this = append(*this, &cookieCopy)
	}
}
// Del removes every cookie with the given name.
func (this *TCookies) Del(name string) {
	cookies := *this
	iCurrent := 0
	iLast := len(cookies) - 1

	for iCurrent <= iLast {
		if cookies[iCurrent] == nil {
			iCurrent++
		} else if cookies[iCurrent].Name == name {
			cookies[iCurrent] = cookies[iLast]
			cookies[iLast] = nil
			iLast--
		} else {
			iCurrent++
		}
	}
	*this = cookies[:iLast+1]
}
// Exist reports whether a cookie with the given name exists.
func (this TCookies) Exist(name string) bool {
	for _, cookie := range this {
		if cookie != nil && cookie.Name == name {
			return true
		}
	}
	return false
}
// Get returns the first cookie with the given name, or nil if none exists.
func (this TCookies) Get(name string) *TCookie {
	for _, cookie := range this {
		if cookie != nil && cookie.Name == name {
			return cookie
		}
	}
	return nil
}
// GetA returns all cookies with the given name.
func (this TCookies) GetAll(name string) []*TCookie {
	var result []*TCookie

	for _, cookie := range this {
		if cookie != nil && cookie.Name == name {
			result = append(result, cookie)
		}
	}
	return result
}

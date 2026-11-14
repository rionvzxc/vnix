package vnix

type TCookies []*TCookie

func (this *TCookies) Set(data TCookies) {
	*this = data
}
func (this *TCookies) Add(cookies ...TCookie) {
	for _, cookie := range cookies {
		cookieCopy := cookie
		*this = append(*this, &cookieCopy)
	}
}
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
func (this TCookies) Exist(name string) bool {
	for _, cookie := range this {
		if cookie != nil && cookie.Name == name {
			return true
		}
	}
	return false
}
func (this TCookies) Get(name string) *TCookie {
	for _, cookie := range this {
		if cookie != nil && cookie.Name == name {
			return cookie
		}
	}
	return nil
}
func (this TCookies) GetA(name string) []*TCookie {
	var result []*TCookie

	for _, cookie := range this {
		if cookie != nil && cookie.Name == name {
			result = append(result, cookie)
		}
	}
	return result
}

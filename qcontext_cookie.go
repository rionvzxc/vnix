package vnix

// Cookie returns the first request cookie value for name or defaultValue.
func (this *QContext) Cookie(name string, defaultValue string) string {
	this.parseCookies()
	return this.reqCookie.GetDefault(name, defaultValue)
}

// CookieValues returns a copy of all request cookie values for name.
func (this *QContext) CookieValues(name string) []string {
	this.parseCookies()
	return this.reqCookie.GetAll(name)
}

// RequestCookie returns a clone of all request cookies.
func (this *QContext) RequestCookie() TValues {
	this.parseCookies()
	return this.reqCookie.Clone()
}

func (this *QContext) parseCookies() {
	if this.reqCookie == nil {
		this.reqCookie = make(TValues)

		for _, cookie := range this.httpRequest.Cookies() {
			this.reqCookie[cookie.Name] = append(this.reqCookie[cookie.Name], cookie.Value)
		}
	}
}

// BindCookie binds request cookies into a struct using cookie tags.
func (this *QContext) BindCookie(value any) error {
	this.parseCookies()
	return bindValues(value, "cookie", this.reqCookie)
}

// SetCookie replaces the response cookie list with a copy of the supplied slice.
func (this *QContext) SetCookie(cookies []*TCookie) {
	this.resCookies = append([]*TCookie(nil), cookies...)
}

// AddCookie appends cookies to the response without replacing existing cookies.
func (this *QContext) AddCookie(cookies []*TCookie) {
	this.resCookies = append(this.resCookies, cookies...)
}

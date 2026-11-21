package vnix

// Header returns the first request header value for name or defaultValue.
func (this *QContext) Header(name string, defaultValue string) string {
	return this.reqHeader.GetDefault(name, defaultValue)
}

// HeaderValues returns a copy of all request header values for name.
func (this *QContext) HeaderValues(name string) []string {
	return this.reqHeader.GetAll(name)
}

// RequestHeader returns a clone of all request headers.
func (this *QContext) RequestHeader() THeader {
	return this.reqHeader.Clone()
}

// SetHeader replaces the response values for name.
func (this *QContext) SetHeader(name string, values ...string) {
	if this.resHeader == nil {
		this.resHeader = make(THeader)
	}
	this.resHeader.Set(name, values...)
}

// AddHeader appends response values for name.
func (this *QContext) AddHeader(name string, values ...string) {
	if this.resHeader == nil {
		this.resHeader = make(THeader)
	}
	this.resHeader.Add(name, values...)
}

// BindHeader binds request headers into a struct using header tags.
func (this *QContext) BindHeader(value any) error {
	return bindValues(value, "header", TValues(this.reqHeader))
}

// SetResponseHeader replaces all response headers with a clone of header.
func (this *QContext) SetResponseHeader(header THeader) {
	this.resHeader = header.Clone()

	if this.resHeader == nil {
		this.resHeader = make(THeader)
	}
}

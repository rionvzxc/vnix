package vnix

import "net/url"

// Query returns the first query parameter value for name or defaultValue.
func (this *QContext) Query(name string, defaultValue string) string {
	if this.reqQuery == nil {
		this.reqQuery = TValues(this.httpRequest.URL.Query())
	}
	return this.reqQuery.GetDefault(name, defaultValue)
}

// QueryValues returns a copy of all query parameter values for name.
func (this *QContext) QueryValues(name string) []string {
	if this.reqQuery == nil {
		this.reqQuery = TValues(this.httpRequest.URL.Query())
	}
	return this.reqQuery.GetAll(name)
}

// RequestQuery returns a clone of all query parameters.
func (this *QContext) RequestQuery() TValues {
	if this.reqQuery == nil {
		this.reqQuery = TValues(this.httpRequest.URL.Query())
	}
	return this.reqQuery.Clone()
}

// BindQuery binds query parameters into a struct using query tags.
func (this *QContext) BindQuery(value any) error {
	values, err := url.ParseQuery(this.httpRequest.URL.RawQuery)

	if err != nil {
		return err
	}
	this.reqQuery = TValues(values)
	return bindValues(value, "query", this.reqQuery)
}

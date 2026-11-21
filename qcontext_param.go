package vnix

// Param returns the first route parameter value for name or defaultValue.
func (this *QContext) Param(name string, defaultValue string) string {
	return this.reqParam.GetDefault(name, defaultValue)
}

// ParamValues returns a copy of all route parameter values for name.
func (this *QContext) ParamValues(name string) []string {
	return this.reqParam.GetAll(name)
}

// RequestParam returns a clone of all route parameters.
func (this *QContext) RequestParam() TValues {
	return this.reqParam.Clone()
}

// BindParam binds route parameters into a struct using uri tags.
func (this *QContext) BindParam(value any) error {
	return bindValues(value, "uri", this.reqParam)
}

package vnix

// Next executes the remaining handlers. Middleware can run code after Next
// returns to process the result of downstream handlers.
func (this *QContext) Next() {
	this.handlerIndex++
	for !this.aborted && this.handlerIndex < len(this.handlers) {
		this.handlers[this.handlerIndex](this)
		this.handlerIndex++
	}
}

// Abort prevents subsequent handlers from running. It does not stop the current
// function or middleware already executing, so return after Abort when needed.
func (this *QContext) Abort() {
	this.aborted = true
}

// IsAborted reports whether Abort has been called for this request.
func (this *QContext) IsAborted() bool {
	return this.aborted
}

// AbortWithStatus sets the response status and aborts the handler chain.
func (this *QContext) AbortWithStatus(code int) {
	this.StatusCode(code)
	this.Abort()
}

// AbortWithStatusJSON sets a JSON response and aborts the handler chain.
func (this *QContext) AbortWithStatusJSON(code int, data any) {
	this.StatusCode(code)
	this.SetJSON(data)
	this.Abort()
}

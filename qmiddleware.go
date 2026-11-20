package vnix

// QMiddleware configures a set of handlers registered on a node.
// By default, handlers are appended and inherited by descendants.
type QMiddleware struct {
	modeSet  bool
	inherit  bool
	handlers []THandlerFunc
}

// ModeAdd appends these handlers after the effective middleware list.
func (this *QMiddleware) ModeAdd() *QMiddleware {
	this.modeSet = false
	return this
}

// ModeSet replaces the effective middleware list, including inherited handlers.
func (this *QMiddleware) ModeSet() *QMiddleware {
	this.modeSet = true
	return this
}

// Inherit controls whether this configuration also applies to descendants.
// When false, descendants continue to inherit the applicable ancestor handlers.
func (this *QMiddleware) Inherit(enabled bool) *QMiddleware {
	this.inherit = enabled
	return this
}

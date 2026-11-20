package vnix

import "net/http"

// TMap represents a JSON-compatible map of values.
type TMap = map[string]any

// TValue represents a map of string values.
type TValue map[string]string

// TCookie aliases the standard library HTTP cookie type.
type TCookie = http.Cookie

// THandlerFunc handles an HTTP request through a QContext.
type THandlerFunc func(*QContext)

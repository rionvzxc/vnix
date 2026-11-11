package vnix

import "net/http"

type TMap = map[string]any
type TValue map[string]string
type TCookie = http.Cookie
type THandlerFunc func(*QContext)
type TNodeType uint
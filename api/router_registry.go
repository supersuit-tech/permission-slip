package api

import "net/http"

// RouteRegistrar is a function that registers HTTP routes on a mux.
// Each API domain file provides one and self-registers it via init().
type RouteRegistrar func(mux *http.ServeMux, deps *Deps)

var routeGroups []RouteRegistrar

// RegisterRouteGroup appends a route registrar to the global registry.
// Call this from an init() function in each domain file.
func RegisterRouteGroup(fn RouteRegistrar) {
	routeGroups = append(routeGroups, fn)
}

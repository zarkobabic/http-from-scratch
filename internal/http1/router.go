package http1

import (
	"log/slog"
	"net/http"
	"strings"
)

type routeRules struct {
	methodHandlers map[string]http.Handler
}

type Router struct {
	routes map[string]routeRules
}

type HandlerFunc func(http.ResponseWriter, *http.Request)

func (f HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f(w, r)
}

func NotAllowedResponse(w http.ResponseWriter, rule routeRules) {
	var allowedMethods []string
	for k := range rule.methodHandlers {
		allowedMethods = append(allowedMethods, k)
	}
	w.Header().Set("Allow", strings.Join(allowedMethods, " "))
	err := writeErrorResponse(w, 405)
	if err != nil {
		slog.Error("method not allowed error:", "error", err)
	}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	urlPath := req.URL.Path
	method := req.Method
	methodHandlers, ok := r.routes[urlPath]
	if !ok {
		err := writeErrorResponse(w, 404)
		if err != nil {
			slog.Error("method not found error:", "error", err)
		}
	}
	handler, ok := methodHandlers.methodHandlers[method]
	if !ok {
		NotAllowedResponse(w, methodHandlers)
	}
	handler.ServeHTTP(w, req)
}

func NewRouter() *Router {
	router := &Router{}
	router.routes = make(map[string]routeRules)
	return router
}

func (r *Router) HandleFunc(method string, pattern string, f func(w http.ResponseWriter, req *http.Request)) {
	rules, ok := r.routes[pattern]
	if !ok {
		methodHandlers := make(map[string]http.Handler)
		rules = routeRules{methodHandlers: methodHandlers}
		r.routes[pattern] = rules
	}
	rules.methodHandlers[method] = HandlerFunc(f)
}

func (r *Router) Handle(method string, pattern string, f http.Handler) {
	rules, ok := r.routes[pattern]
	if !ok {
		methodHandlers := make(map[string]http.Handler)
		rules = routeRules{methodHandlers: methodHandlers}
		r.routes[pattern] = rules
	}
	rules.methodHandlers[method] = f
}

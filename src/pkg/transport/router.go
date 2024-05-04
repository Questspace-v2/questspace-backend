package transport

import (
	"net/http"
	"slices"

	"github.com/julienschmidt/httprouter"
)

type Middleware func(next http.Handler) http.Handler

type RouteHandler interface {
	Use(middlewares ...Middleware) RouteHandler
	GET(path string, handlerFunc http.Handler)
	POST(path string, handlerFunc http.Handler)
	PUT(path string, handlerFunc http.Handler)
	DELETE(path string, handlerFunc http.Handler)
	PATCH(path string, handlerFunc http.Handler)
	OPTIONS(path string, handlerFunc http.Handler)
}

type routeHandler struct {
	mw  []Middleware
	mux *httprouter.Router
}

type Router struct {
	mw  []Middleware
	mux *httprouter.Router
}

func NewRouter() *Router {
	return &Router{
		mux: httprouter.New(),
	}
}

func (r *Router) Use(middlewares ...Middleware) *Router {
	r.mw = append(r.mw, middlewares...)
	return r
}

func (r *Router) H() RouteHandler {
	return &routeHandler{
		mw:  slices.Clone(r.mw),
		mux: r.mux,
	}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

func (r *routeHandler) Use(middlewares ...Middleware) RouteHandler {
	r.mw = append(r.mw, middlewares...)
	return r
}

func (r *routeHandler) serve(method, path string, h http.Handler) {
	handler := h
	for i := len(r.mw) - 1; i >= 0; i-- {
		handler = r.mw[i](handler)
	}
	r.mux.Handler(method, path, handler)
}

func (r *routeHandler) GET(path string, h http.Handler) {
	r.serve(http.MethodGet, path, h)
}

func (r *routeHandler) POST(path string, h http.Handler) {
	r.serve(http.MethodPost, path, h)
}

func (r *routeHandler) PUT(path string, h http.Handler) {
	r.serve(http.MethodPut, path, h)
}

func (r *routeHandler) DELETE(path string, h http.Handler) {
	r.serve(http.MethodDelete, path, h)
}

func (r *routeHandler) PATCH(path string, h http.Handler) {
	r.serve(http.MethodPatch, path, h)
}

func (r *routeHandler) OPTIONS(path string, h http.Handler) {
	r.serve(http.MethodOptions, path, h)
}

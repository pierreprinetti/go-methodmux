/*
Package methodmux provides a method-aware HTTP router based on net/http.

Example usage:

	package main

	import (
		"net/http"

		methodmux "github.com/pierreprinetti/go-methodmux"
	)

	func main() {
		mux := methodmux.New()

		mux.Handle(http.MethodGet, "/some-endpoint/", getHandler)
		mux.Handle("MYMETHOD", "/some-endpoint/", myMethodHandler)
		mux.HandleFunc(http.MethodPost, "/some-endpoint/", func(rw http.ResponseWriter, req *http.Request) {
			fmt.Fprintf(rw, "Hi! This the response to a POST call.")
		})

		srv := &http.Server{
			Handler:        mux,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}

		log.Fatal(srv.ListenAndServe())
	}

Methodmux exposes a single type: `ServeMux`. `ServeMux` holds a separate `http.ServeMux` for every HTTP verb an http.Handler has been registered to.

Every new request will be matched against the underlying `http.ServeMux` that corresponds to the HTTP method of the request.
If no match is found, `ServeMux` will look for a match in the other HTTP verbs. If a match is found, an HTTP code 405 "Method Not Allowed" is returned. If not, an HTTP code 404 "Not Found" is returned.

Methodmux has been written with readability in mind and is just as fast and efficient as `net/http` is.
*/
package methodmux // import "github.com/pierreprinetti/go-methodmux"

import (
	"net/http"
	"sync"
)

var (
	// BadRequestHandler is a http.Handler that replies to the
	// request with an HTTP 400 "Bad Request" error.
	BadRequestHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	})

	// NotFoundHandler is a http.Handler that replies to the request with
	// an HTTP 404 "Not Found" error.
	NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	})

	// MethodNotAllowedHandler is a http.Handler that replies to
	// the request with an HTTP 405 "Method Not Allowed" error.
	MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	})
)

// ServeMux is a method-aware HTTP request multiplexer.
// Every registered handler will be only served for the particular HTTP method
// it has been registered with.
type ServeMux struct {
	mu sync.RWMutex
	m  map[string]*http.ServeMux
}

// New allocates and returns a new ServeMux.
func New() *ServeMux {
	return new(ServeMux)
}

// Handle registers the handler for the given method and pattern.
// If a handler already exists for the combination of method and pattern, Handle panics.
// The documentation for http.ServeMux explains how patterns are matched.
func (mux *ServeMux) Handle(method, pattern string, handler http.Handler) {
	mux.mu.Lock()
	defer mux.mu.Unlock()

	if mux.m == nil {
		mux.m = make(map[string]*http.ServeMux)
	}

	if _, exists := mux.m[method]; !exists {
		mux.m[method] = http.NewServeMux()
	}

	mux.m[method].Handle(pattern, handler)
}

// HandleFunc registers the handler function for the given method and pattern.
func (mux *ServeMux) HandleFunc(method, pattern string, handler func(http.ResponseWriter, *http.Request)) {
	mux.Handle(method, pattern, http.HandlerFunc(handler))
}

// Handler returns the handler to use for the given request,
// consulting r.Method, r.Host, and r.URL.Path. It always returns
// a non-nil handler. If the path is not in its canonical form, the
// handler will be an internally-generated handler that redirects
// to the canonical path. If the host contains a port, it is ignored
// when matching handlers.
//
// The path and host are used unchanged for CONNECT requests.
//
// Handler also returns the registered pattern that matches the
// request or, in the case of internally-generated redirects,
// the pattern that will match after following the redirect.
//
// If there is no registered handler that applies to the request,
// Handler checks the other methods on the same pattern.
// If the same pattern matches with a handle that responds to another
// HTTP method, a "Method Not Allowed" handler is returned with an
// empty pattern. If no HTTP method would trigger a registered
// handler, "Not Found" handler is returned with an empty pattern.
func (mux *ServeMux) Handler(r *http.Request) (h http.Handler, pattern string) {
	mux.mu.RLock()
	defer mux.mu.RUnlock()

	if _, exists := mux.m[r.Method]; exists {
		h, pattern = mux.m[r.Method].Handler(r)
	}

	if pattern == "" {
		for _, mux := range mux.m {
			if _, crossMethodPattern := mux.Handler(r); crossMethodPattern != "" {
				return MethodNotAllowedHandler, ""
			}
		}
		return NotFoundHandler, ""
	}

	return h, pattern
}

// ServeHTTP dispatches the request to the handler registered
// with the HTTP method of the request, and whose pattern most
// closely matches the request URL.
// If no registered matcher is found, a 405 is returned if there
// is a match with another HTTP method. Otherwise, a 404 is returned.
func (mux *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.RequestURI == "*" {
		if r.ProtoAtLeast(1, 1) {
			w.Header().Set("Connection", "close")
		}
		BadRequestHandler.ServeHTTP(w, r)
		return
	}

	h, _ := mux.Handler(r)
	h.ServeHTTP(w, r)
}

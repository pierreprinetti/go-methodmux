# Methodmux
[![GoDoc](https://godoc.org/github.com/pierreprinetti/go-methodmux?status.svg)](http://godoc.org/github.com/pierreprinetti/go-methodmux)
[![Build Status](https://travis-ci.org/pierreprinetti/go-methodmux.svg?branch=master)](https://travis-ci.org/pierreprinetti/go-methodmux)
[![Go Report Card](https://goreportcard.com/badge/github.com/pierreprinetti/go-methodmux)](https://goreportcard.com/report/github.com/pierreprinetti/go-methodmux)
[![Coverage Status](https://coveralls.io/repos/github/pierreprinetti/go-methodmux/badge.svg?branch=master)](https://coveralls.io/github/pierreprinetti/go-methodmux?branch=master)

Methodmux is a method-aware HTTP router based on net/http.

Methodmux exposes a single type: `ServeMux`. `ServeMux` holds a separate `http.ServeMux` for every HTTP verb an http.Handler has been registered to.

Every new request will be matched against the underlying `http.ServeMux` that corresponds to the HTTP method of the request.
If no match is found, `ServeMux` will look for a match in the other HTTP verbs. If a match is found, an HTTP code 405 "Method Not Allowed" is returned. If not, an HTTP code 404 "Not Found" is returned.

Methodmux has been written with readability in mind and is just as fast and efficient as `net/http` is.

## API

* `func New() *ServeMux`: allocates and returns a new ServeMux.
* `func (mux *ServeMux) Handle(method, pattern string, handler http.Handler)`: registers the handler for the given method and pattern.
* `func (mux *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request)`: dispatches the request to the handler registered with the HTTP method of the request, and whose pattern most closely matches the request URL.

## Usage

```Go
package main

import (
	"net/http"

	methodmux "github.com/pierreprinetti/go-methodmux"
)

func main() {
	mux := methodmux.New()

	getHandler := myNewGetHandler()

	mux.Handle(http.MethodGet, "/some-endpoint/", getHandler)
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
```

Use godoc for more detailed documentation.

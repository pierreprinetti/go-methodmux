package methodmux_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	. "github.com/pierreprinetti/go-methodmux"
)

// serve returns a handler that sends a response with the given code.
func serve(code int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
	}
}

func TestServe(t *testing.T) {
	testCases := [...]struct {
		method          string
		host            string
		path            string
		expectedCode    int
		expectedPattern string
	}{
		{"GET", "example.com", "/", 404, ""},
		{"GET", "example.com", "/dir", 301, "/dir/"},
		{"GET", "example.com", "/dir/", 200, "/dir/"},
		{"GET", "example.com", "/dir/file", 200, "/dir/"},
		{"GET", "example.com", "/search", 201, "/search"},
		{"GET", "example.com", "/search/", 404, ""},
		{"GET", "example.com", "/search/foo", 404, ""},
		{"GET", "sub.example.com", "/search", 202, "sub.example.com/search"},
		{"GET", "sub.example.com", "/search/", 203, "sub.example.com/"},
		{"GET", "sub.example.com", "/search/foo", 203, "sub.example.com/"},
		{"GET", "sub.example.com", "/", 203, "sub.example.com/"},
		{"GET", "sub.example.com:443", "/", 203, "sub.example.com/"},
		{"GET", "images.example.com", "/search", 201, "/search"},
		{"GET", "images.example.com", "/search/", 404, ""},
		{"GET", "images.example.com", "/search/foo", 404, ""},
		{"GET", "example.com", "/../search", 301, "/search"},
		{"GET", "example.com", "/dir/./file", 301, "/dir/"},

		{"POST", "example.com", "/", 404, ""},
		{"POST", "example.com", "/dir", 301, "/dir/"},
		{"POST", "example.com", "/dir/", 204, "/dir/"},
		{"POST", "example.com", "/dir/file", 204, "/dir/"},
		{"POST", "example.com", "/search", 205, "/search"},
		{"POST", "example.com", "/search/", 404, ""},
		{"POST", "example.com", "/search/foo", 404, ""},
		{"POST", "sub.example.com", "/search", 206, "sub.example.com/search"},
		{"POST", "sub.example.com", "/search/", 208, "sub.example.com/"},
		{"POST", "sub.example.com", "/search/foo", 208, "sub.example.com/"},
		{"POST", "sub.example.com", "/", 208, "sub.example.com/"},
		{"POST", "sub.example.com:443", "/", 208, "sub.example.com/"},
		{"POST", "images.example.com", "/search", 205, "/search"},
		{"POST", "images.example.com", "/search/", 404, ""},
		{"POST", "images.example.com", "/search/foo", 404, ""},
		{"POST", "example.com", "/../search", 301, "/search"},
		{"POST", "example.com", "/dir/./file", 301, "/dir/"},

		{"PATCH", "example.com", "/", 404, ""},
		{"PATCH", "example.com", "/dir", 301, "/dir/"},
		{"PATCH", "example.com", "/dir/", 413, "/dir/"},
		{"PATCH", "example.com", "/dir/file", 413, "/dir/"},
		{"PATCH", "example.com", "/search", 414, "/search"},
		{"PATCH", "example.com", "/search/", 404, ""},
		{"PATCH", "example.com", "/search/foo", 404, ""},
		{"PATCH", "sub.example.com", "/search", 415, "sub.example.com/search"},
		{"PATCH", "sub.example.com", "/search/", 416, "sub.example.com/"},
		{"PATCH", "sub.example.com", "/search/foo", 416, "sub.example.com/"},
		{"PATCH", "sub.example.com", "/", 416, "sub.example.com/"},
		{"PATCH", "sub.example.com:443", "/", 416, "sub.example.com/"},
		{"PATCH", "images.example.com", "/search", 414, "/search"},
		{"PATCH", "images.example.com", "/search/", 404, ""},
		{"PATCH", "images.example.com", "/search/foo", 404, ""},
		{"PATCH", "example.com", "/../search", 301, "/search"},
		{"PATCH", "example.com", "/dir/./file", 301, "/dir/"},

		{"GET", "example.com", "/get-only", 418, "/get-only"},
		{"POST", "example.com", "/get-only", 405, ""},
		{"PATCH", "example.com", "/get-only", 405, ""},

		// The /foo -> /foo/ redirect applies to CONNECT requests
		// but the path canonicalization does not.
		{"CONNECT", "example.com", "/dir", 301, "/dir/"},
		{"CONNECT", "example.com", "/../search", 404, ""},
		{"CONNECT", "example.com", "/dir/..", 200, "/dir/"},
		{"CONNECT", "example.com", "/dir/./file", 200, "/dir/"},
	}

	var register = [...]struct {
		method  string
		pattern string
		h       http.Handler
	}{
		{"GET", "/dir/", serve(200)},
		{"GET", "/search", serve(201)},
		{"GET", "sub.example.com/search", serve(202)},
		{"GET", "sub.example.com/", serve(203)},
		{"GET", "/get-only", serve(418)},
		{"POST", "/dir/", serve(204)},
		{"POST", "/search", serve(205)},
		{"POST", "sub.example.com/search", serve(206)},
		{"POST", "sub.example.com/", serve(208)},
		{"PATCH", "/dir/", serve(413)},
		{"PATCH", "/search", serve(414)},
		{"PATCH", "sub.example.com/search", serve(415)},
		{"PATCH", "sub.example.com/", serve(416)},
		{"CONNECT", "/dir/", serve(200)},
		{"CONNECT", "/search", serve(201)},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s %s%s", tc.method, tc.host, tc.path), func(t *testing.T) {
			mux := New()

			for _, e := range register {
				mux.Handle(e.method, e.pattern, e.h)
			}

			r := &http.Request{
				Method: tc.method,
				Host:   tc.host,
				URL: &url.URL{
					Path: tc.path,
				},
			}
			h, pattern := mux.Handler(r)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, r)
			if have, want := rr.Code, tc.expectedCode; have != want {
				t.Errorf("expected status code %d, found %d", want, have)
			}
			if have, want := pattern, tc.expectedPattern; have != want {
				t.Errorf("expected canonicalized pattern %q, found %q", want, have)
			}
		})
	}
}

func TestServeHTTP(t *testing.T) {
	t.Run("returns a handler", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/some/path", nil)
		rw := &httptest.ResponseRecorder{}
		s := New()
		s.Handle("DELETE", "/some/path", serve(303))
		s.ServeHTTP(rw, req)
		if want, have := 303, rw.Code; have != want {
			t.Errorf("expected status code %d, found %d", want, have)
		}
	})

	t.Run("bad request on *", func(t *testing.T) {
		req := httptest.NewRequest("GET", "*", nil)
		rw := &httptest.ResponseRecorder{}
		New().ServeHTTP(rw, req)
		res := rw.Result()
		if want, have := 400, res.StatusCode; have != want {
			t.Errorf("expected status code %d, found %d", want, have)
		}
		if want, have := "close", res.Header.Get("Connection"); have != want {
			t.Errorf("expected \"Connection: %s\" header, found %q", want, have)
		}
	})
}

func TestHandleFunc(t *testing.T) {
	t.Run("registers a handler", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/some/path", nil)
		rw := &httptest.ResponseRecorder{}
		s := New()
		s.HandleFunc("DELETE", "/some/path", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(512)
		})
		s.ServeHTTP(rw, req)
		if want, have := 512, rw.Code; have != want {
			t.Errorf("expected status code %d, found %d", want, have)
		}
	})
}

func BenchmarkServeMux(b *testing.B) {
	type test struct {
		method string
		path   string
		code   int
		req    *http.Request
	}

	// Build example handlers and requests
	var tests []test
	methods := []string{"GET", "POST", "PATCH"}
	endpoints := []string{"search", "dir", "file", "change", "count", "s"}
	for _, m := range methods {
		for _, e := range endpoints {
			for i := 200; i < 210; i++ {
				p := fmt.Sprintf("/%s/%d/", e, i)
				tests = append(tests, test{
					method: m,
					path:   p,
					code:   i,
					req:    &http.Request{Method: m, Host: "localhost", URL: &url.URL{Path: p}},
				})
			}
		}
	}
	mux := New()
	for _, tt := range tests {
		mux.Handle(tt.method, tt.path, serve(tt.code))
	}

	rw := httptest.NewRecorder()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tt := range tests {
			*rw = httptest.ResponseRecorder{}
			h, pattern := mux.Handler(tt.req)
			h.ServeHTTP(rw, tt.req)
			if pattern != tt.path || rw.Code != tt.code {
				b.Fatalf("got %d, %q, want %d, %q", rw.Code, pattern, tt.code, tt.path)
			}
		}
	}
}

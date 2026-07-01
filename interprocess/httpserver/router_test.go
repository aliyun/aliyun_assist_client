package httpserver

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockHTTPResponseWriter struct {
	statusCode int
	header     http.Header
	body       []byte
}

func (m *mockHTTPResponseWriter) Header() http.Header {
	return m.header
}
func (m *mockHTTPResponseWriter) Write(b []byte) (int, error) {
	m.body = append(m.body, b...)
	return len(b), nil
}
func (m *mockHTTPResponseWriter) WriteHeader(statusCode int) {
	m.statusCode = statusCode
	return
}

type readCloser struct {
	bytes.Buffer
}

func (r readCloser) Close() error {
	return nil
}

func Test_TestHandler(t *testing.T) {
	testCases := []struct {
		name         string
		method       string
		url          string
		expectedPath string
		headers      []string
		body         string
	}{
		{
			name:         "case1",
			method:       "POST",
			url:          "http://localhost/test",
			expectedPath: "/test",
			headers: []string{
				"K1: v1,v12",
				"K2: v3",
			},
			body: "body of test",
		},
	}

	resetMux()
	combineHeaders := func(headers []string) string {
		return strings.Join(headers, "\n")
	}
	handler := SetupRouter()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf := bytes.NewBuffer([]byte(tc.body))
			rc := readCloser{*buf}
			r := &http.Request{
				Method:     tc.method,
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				Header:     make(http.Header),
				Body:       &rc,
			}
			URL, err := url.ParseRequestURI(tc.url)
			assert.Nil(t, err)
			r.URL = URL
			for _, line := range tc.headers {
				kv := strings.SplitN(line, ": ", 2)
				k, v := kv[0], kv[1]
				for _, vv := range strings.Split(v, ",") {
					r.Header.Add(k, vv)
				}
			}

			w := &mockHTTPResponseWriter{
				header: make(http.Header),
				body:   make([]byte, 0),
			}

			handler.ServeHTTP(w, r)
			expectedResp := fmt.Sprintf("%s\n%s\n%s\n%s\n\n%s\n", tc.method, tc.expectedPath, "HTTP/1.1", combineHeaders(tc.headers), tc.body)
			assert.Equal(t, expectedResp, string(w.body))
			fmt.Println(string(w.body))
		})
	}
}

func TestRPCHandler(t *testing.T) {
	testCases := []struct {
		name               string
		method             string
		url                string
		proto              string
		headers            map[string][]string
		expectedStatusCode int
	}{
		{
			name:   "normal case",
			method: "POST",
			url:    "http://localhost/test/rpc",
			proto:  "HTTP/1.1",
			headers: map[string][]string{
				"Content-Type": {"application/json"},
				"Accept":       {"application/json"},
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:   "bad method",
			method: "GET",
			url:    "http://localhost/test/rpc",
			proto:  "HTTP/1.1",
			headers: map[string][]string{
				"Content-Type": {"application/json"},
				"Accept":       {"application/json"},
			},
			expectedStatusCode: http.StatusMethodNotAllowed,
		},
		{
			name:   "bad Content-Type",
			method: "POST",
			url:    "http://localhost/test/rpc",
			proto:  "HTTP/1.1",
			headers: map[string][]string{
				"Content-Type": {"application/text"},
				"Accept":       {"application/json"},
			},
			expectedStatusCode: http.StatusUnsupportedMediaType,
		},
		{
			name:   "bad Accept",
			method: "POST",
			url:    "http://localhost/test/rpc",
			proto:  "HTTP/1.1",
			headers: map[string][]string{
				"Content-Type": {"application/json"},
				"Accept":       {"application/text"},
			},
			expectedStatusCode: http.StatusUnsupportedMediaType,
		},
	}

	resetMux()
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	RegisterHTTPHandler("/test/rpc", mockHandler)
	handler := SetupRouter()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := &http.Request{
				Method:     tc.method,
				Proto:      tc.proto,
				ProtoMajor: 1,
				ProtoMinor: 1,
				Header:     tc.headers,
			}
			URL, err := url.ParseRequestURI(tc.url)
			assert.Nil(t, err)
			r.URL = URL

			w := &mockHTTPResponseWriter{
				header: make(http.Header),
				body:   make([]byte, 0),
			}

			handler.ServeHTTP(w, r)
			assert.Equal(t, tc.expectedStatusCode, w.statusCode)
		})
	}
}

func resetMux() {
	mux = http.NewServeMux()
	muxAlreadyInUsed = false
}

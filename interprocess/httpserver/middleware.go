package httpserver

import (
	"net/http"
	"strings"
)

type MidwareRPC struct{}

func (m *MidwareRPC) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	// Do pre-check for */rpc requests
	if strings.HasSuffix(r.URL.Path, "/rpc") {
		if r.Method != "POST" {
			rw.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Content-Type") != "application/json" {
			rw.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}
		if r.Header.Get("Accept") != "application/json" {
			rw.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}

		defer func() {
			rw.Header().Set("Content-Type", "application/json")
		}()

	}
	next(rw, r)
}

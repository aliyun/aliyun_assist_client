package httpserver

import (
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/urfave/negroni/v3"
)

var (
	mux              = http.NewServeMux()
	muxLock          sync.Mutex
	muxAlreadyInUsed bool
)

func SetupRouter() http.Handler {
	muxLock.Lock()
	defer muxLock.Unlock()

	if muxAlreadyInUsed {
		panic("mux already in used")
	}
	mux.HandleFunc("/test", testHandler)
	muxAlreadyInUsed = true
	n := negroni.New()
	n.Use(negroni.NewRecovery())
	n.Use(&MidwareRPC{})

	n.UseHandler(mux)
	return n
}

func RegisterHTTPHandler(pattern string, handler http.HandlerFunc) {
	muxLock.Lock()
	defer muxLock.Unlock()

	if muxAlreadyInUsed {
		panic("mux already in used")
	}

	mux.HandleFunc(pattern, handler)
}

// The test handler is used to verify the connectivity of the HTTP interface.
// Do not add handlers for other interfaces here.
func testHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(r.Method))
	w.Write([]byte{'\n'})
	w.Write([]byte(r.URL.Path))
	w.Write([]byte{'\n'})
	w.Write([]byte(r.Proto))
	w.Write([]byte{'\n'})
	for k, v := range r.Header {
		w.Write([]byte(k))
		w.Write([]byte(": "))
		w.Write([]byte(strings.Join(v, ",")))
		w.Write([]byte{'\n'})
	}
	w.Write([]byte{'\n'})
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.Write([]byte(err.Error()))
	} else {
		w.Write(body)
	}
	w.Write([]byte{'\n'})
}

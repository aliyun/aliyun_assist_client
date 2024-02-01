package wrapgo

import (
	"runtime/debug"
	"sync"
	"fmt"
	"os"
)

// PanicHandler is used in wrapped goroutine initiation
type PanicHandler func(interface{}, []byte)

var (
	_defaultPanicHandler PanicHandler = defaultPanicHandler
	_defaultPanicHandlerInitLock sync.Mutex
)

func defaultPanicHandler(payload interface{}, stack []byte) {
	panic(payload)
}

// SetDefaultPanicHandler sets default panic handler for GoWithDefaultPanicHandler.
// It should be called at initialization only once.
func SetDefaultPanicHandler(handler PanicHandler) PanicHandler {
	_defaultPanicHandlerInitLock.Lock()
	defer _defaultPanicHandlerInitLock.Unlock()

	oldPanicHandler := _defaultPanicHandler
	_defaultPanicHandler = handler
	return oldPanicHandler
}

// CallDefaultPanicHandler calls set defaultPanicHandler function as wrapper
func CallDefaultPanicHandler(payload interface{}, stack []byte) {
	_defaultPanicHandler(payload, stack)
}

// GoWithPanicHandler initiate goroutine with panic handler
func GoWithPanicHandler(f func(), handler PanicHandler) {
	go CallWithPanicHandler(f, handler)
}

// CallWithPanicHandler call f with panic handler
func CallWithPanicHandler(f func(), handler PanicHandler) {
	defer func () {
		if panicPayload := recover(); panicPayload != nil {
			stacktrace := debug.Stack()
			fmt.Fprintf(os.Stderr, "panic: %v", panicPayload)
			fmt.Fprint(os.Stderr, string(stacktrace))
			handler(panicPayload, stacktrace)
		}
	}()

	f()
}

// GoWithDefaultPanicHandler initiate goroutine with default panic handler
func GoWithDefaultPanicHandler(f func()) {
	GoWithPanicHandler(f, _defaultPanicHandler)
}

// CallWithPanicHandler call f with panic handler
func CallWithDefaultPanicHandler(f func()) {
	CallWithPanicHandler(f, _defaultPanicHandler)
}

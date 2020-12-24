package wrapgo

import (
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPackageDefaultHandler(t *testing.T) {
	const PanicMessage = "Panic just here"
	assert.PanicsWithValue(t, PanicMessage, func ()  {
		defaultPanicHandler(PanicMessage, []byte{})
	}, "Object thrown outside should be specified panic message")
}

func TestSetDefaultPanicHandler(t *testing.T) {
	panicNoop := func(payload interface{}, stacktrace []byte) {}

	// 1. Test SetDefaultPanicHandler bahavior without assumption of existing panic handler
	oldPanicHandler := SetDefaultPanicHandler(panicNoop)
	assert.Exactly(t, reflect.ValueOf(panicNoop).Pointer(),
		reflect.ValueOf(_defaultPanicHandler).Pointer(),
		"defaultPanicHanlder should be set with panicNoop")

	// 2. Test SetDefaultPanicHandler bahavior with assumption of existing panic handler as panicNoop,
	// which should set default panic handler with oldPanicHandler
	assert.Exactly(t, reflect.ValueOf(panicNoop).Pointer(),
		reflect.ValueOf(SetDefaultPanicHandler(oldPanicHandler)).Pointer(),
		"SetDefaultPanicHandler should return panicNoop as old panic handler")

	// 3. Test SetDefaultPanicHandler bahavior via validating _defaultPanicHandler variable
	assert.Exactly(t, reflect.ValueOf(oldPanicHandler).Pointer(),
		reflect.ValueOf(_defaultPanicHandler).Pointer(),
		"defaultPanicHanlder should be set with oldPanicHandler")
}

func TestCallDefaultPanicHandler(t *testing.T) {
	const PanicMessage = "Panic just here"
	assert.PanicsWithValue(t, PanicMessage, func ()  {
		CallDefaultPanicHandler(PanicMessage, []byte{})
	}, "Object thrown outside should be specified panic message")
}

func TestCallCustomDefaultPanicHandler(t *testing.T) {
	const PanicMessage = "Panic just here"
	var thrownOutside interface{}
	oldPanicHandler := SetDefaultPanicHandler(func(payload interface{}, stacktrace []byte) {
		thrownOutside = payload
	})
	defer func() {
		SetDefaultPanicHandler(oldPanicHandler)
	}()

	CallDefaultPanicHandler(PanicMessage, []byte{})
	assert.Exactly(t, string(PanicMessage), thrownOutside, "Object thrown outside should be specified panic message")
}

func TestGoWithPanicHandler(t *testing.T) {
	const PanicMessage = "Panic here thrown outside"
	var thrownOutside interface{}
	var wg sync.WaitGroup

	wg.Add(1)
	GoWithPanicHandler(func() {
		panic(PanicMessage)
	}, func(payload interface{}, stacktrace []byte) {
		thrownOutside = payload
		wg.Done()
	})
	wg.Wait()

	assert.Exactly(t, string(PanicMessage), thrownOutside, "Object thrown outside should be specified panic message")
}

func TestGoWithCustomDefaultPanicHandler(t *testing.T) {
	const PanicMessage = "Panic here thrown outside"
	var thrownOutside interface{}
	var wg sync.WaitGroup

	wg.Add(1)
	oldPanicHandler := SetDefaultPanicHandler(func(payload interface{}, stacktrace []byte) {
		thrownOutside = payload
		wg.Done()
	})
	defer func() {
		SetDefaultPanicHandler(oldPanicHandler)
	}()

	GoWithDefaultPanicHandler(func() {
		panic(PanicMessage)
	})
	wg.Wait()

	assert.Exactly(t, string(PanicMessage), thrownOutside, "Object thrown outside should be specified panic message")
}

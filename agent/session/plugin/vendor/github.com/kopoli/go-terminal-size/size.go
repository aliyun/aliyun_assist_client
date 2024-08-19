// Package tsize gets the terminal size. Supports Linux and Windows.
package tsize

import (
	"errors"
	"os"

	isatty "github.com/mattn/go-isatty"
)

// Size represents terminal size in columns and rows as Width and Height,
// respectively.
type Size struct {
	Width  int
	Height int
}

var isTerminal = isatty.IsTerminal

// ErrNotATerminal is the error to return if the given file to FgetSize isn't a
// terminal.
var ErrNotATerminal = errors.New("Given file is not a terminal")

// GetSize gets the current terminal size. The terminal is expected to be
// os.Stdout. Returns the NotATerminal error, if it is not a terminal.
func GetSize() (s Size, err error) {
	return FgetSize(os.Stdout)
}

// FgetSize gets the terminal size of a given os.File. Returns the NotATerminal error, if it is not a terminal.
func FgetSize(fp *os.File) (s Size, err error) {
	if fp == nil || !isTerminal(fp.Fd()) {
		err = ErrNotATerminal
		return
	}

	s, err = getTerminalSize(fp)
	return
}

// SizeListener listens to terminal size changes. The new size is returned
// through the Change channel when the change occurs.
type SizeListener struct {
	Change <-chan Size

	done chan struct{}
}

// Close implements the io.Closer interface that stops listening to terminal
// size changes.
func (sc *SizeListener) Close() (err error) {
	if sc.done != nil {
		close(sc.done)
		sc.done = nil
		sc.Change = nil
	}

	return
}

// NewSizeListener creates a new size change listener
func NewSizeListener() (sc *SizeListener, err error) {
	sc = &SizeListener{}

	sizechan := make(chan Size, 1)
	sc.Change = sizechan
	sc.done = make(chan struct{})

	err = getTerminalSizeChanges(sizechan, sc.done)
	if err != nil {
		close(sizechan)
		close(sc.done)
		sc = &SizeListener{}
		return
	}

	return
}

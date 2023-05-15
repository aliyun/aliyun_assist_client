# Go terminal size

[![GoDoc](https://godoc.org/github.com/kopoli/go-terminal-size?status.svg)](https://godoc.org/github.com/kopoli/go-terminal-size)
[![Build Status](https://travis-ci.org/kopoli/go-terminal-size.svg?branch=master)](https://travis-ci.org/kopoli/go-terminal-size)
[![Go Report Card](https://goreportcard.com/badge/github.com/kopoli/go-terminal-size)](https://goreportcard.com/report/github.com/kopoli/go-terminal-size)

Features:
- Get the size of the current terminal as rows and columns.
- Listen on terminal size changes and receive the new size via a channel.
- Supports Linux and Windows.

## Installation

```
$ go get github.com/kopoli/go-terminal-size
```

## Usage

For a complete example see `_example/example.go`.

Abbreviated example:
```golang
package main

import (
	"fmt"

	tsize "github.com/kopoli/go-terminal-size"
)

func main() {
	var s tsize.Size

	s, err := tsize.GetSize()
	if err == nil {
		fmt.Println("Current size is", s.Width, "by", s.Height)
	}
}
```

## License

MIT license

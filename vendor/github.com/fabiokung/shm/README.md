## POSIX shared memory

Portable across (most?) UNIX flavors: linux, freebsd and darwin currently.

### Usage

```go
package main

import (
	"os"

	"github.com/fabiokung/shm"
)

func main() {
	file, err := shm.Open("my_region", os.O_RDRW|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	// syscall.Ftruncate if new, etc
	defer file.Close()
	defer shm.Unlink(file.Name())
}
```

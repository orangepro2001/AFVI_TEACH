//go:build !windows

// Command setup is the Windows installer; there is nothing to run on other systems. The file only
// lets go vet ./... and go build ./... succeed on Linux.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "setup runs on Windows only; build it with GOOS=windows (scripts/package-wsl.sh)")
	os.Exit(2)
}

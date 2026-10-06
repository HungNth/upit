//go:build !windows

package main

import (
	"fmt"
	"os"
)

var launcherFormat = "1"

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--launcher-format" {
		fmt.Println(launcherFormat)
		os.Exit(0)
	}
	fmt.Fprintln(os.Stderr, "upit launcher is only supported on Windows")
	os.Exit(1)
}

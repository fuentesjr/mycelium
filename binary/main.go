package main

import (
	"os"
)

func main() {
	os.Exit(dispatch(os.Stdin, os.Stdout, os.Stderr, os.Args[1:]))
}

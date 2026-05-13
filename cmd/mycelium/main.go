package main

import (
	"os"

	"mycelium/internal/mycelium"
)

func main() {
	os.Exit(mycelium.Dispatch(os.Stdin, os.Stdout, os.Stderr, os.Args[1:]))
}

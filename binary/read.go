package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
)

func readFile(out, errOut io.Writer, mount, requested string) int {
	abs, err := resolveUnderMount(mount, requested)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium read: %v\n", err)
		if errors.Is(err, ErrMountUnset) {
			return ExitGenericError
		}
		return ExitUsage
	}
	f, err := os.Open(abs)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			fmt.Fprintf(errOut, "mycelium read: %s: not found\n", requested)
			return ExitGenericError
		}
		fmt.Fprintf(errOut, "mycelium read: %v\n", err)
		return ExitGenericError
	}
	defer f.Close()
	if _, err := io.Copy(out, f); err != nil {
		fmt.Fprintf(errOut, "mycelium read: %v\n", err)
		return ExitGenericError
	}
	return ExitOK
}

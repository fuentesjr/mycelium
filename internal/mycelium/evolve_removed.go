package mycelium

import (
	"fmt"
	"io"
)

func runEvolve(_ io.Reader, _ io.Writer, errOut io.Writer, _ []string) int {
	fmt.Fprintln(errOut, "evolve was removed in 0.4.0; record conventions in your conventions file (see MYCELIUM_MEMORY.md)")
	return ExitGenericError
}

package main

import (
	"fmt"
	"io"
)

const (
	stubLogResponse = `{"log_status":"ok"}` + "\n"
)

func stubGrep() int {
	return ExitOK
}

func stubLogOrRemove(out io.Writer) int {
	fmt.Fprint(out, stubLogResponse)
	return ExitOK
}

package main

import (
	"fmt"
	"io"
)

const (
	stubMutationResponse = `{"version":"sha256:stubbed","log_status":"ok"}` + "\n"
	stubLogResponse      = `{"log_status":"ok"}` + "\n"
)

func stubWriteOrEdit(out io.Writer) int {
	fmt.Fprint(out, stubMutationResponse)
	return ExitOK
}

func stubGrep() int {
	return ExitOK
}

func stubLogOrRemove(out io.Writer) int {
	fmt.Fprint(out, stubLogResponse)
	return ExitOK
}

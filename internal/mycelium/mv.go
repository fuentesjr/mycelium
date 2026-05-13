package mycelium

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"unicode/utf8"
)

func emitDestinationExists(errOut io.Writer, mount, dstAbs string, includeContent bool, rationale string) int {
	dstRel := relForwardSlash(mount, dstAbs)
	dstVer, verErr := currentVersion(dstAbs)
	if verErr != nil {
		fmt.Fprintf(errOut, "mycelium mv: %v\n", verErr)
		return ExitGenericError
	}
	env := conflictEnvelope{
		Error:          "destination_exists",
		Op:             "mv",
		Path:           dstRel,
		CurrentVersion: dstVer,
		Rationale:      rationale,
	}
	if includeContent {
		fileBytes, readErr := os.ReadFile(dstAbs)
		if readErr == nil && utf8.Valid(fileBytes) {
			env.CurrentContent = string(fileBytes)
		}
	}
	line, _ := json.Marshal(env)
	line = append(line, '\n')
	_, _ = errOut.Write(line)
	return ExitConflict
}

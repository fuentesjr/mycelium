package mycelium

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"unicode/utf8"
)

type readEnvelope struct {
	Path    string `json:"path"`
	Version string `json:"version"`
	Content string `json:"content"`
}

func readFile(out, errOut io.Writer, mount, requested, format string) int {
	if format != "text" && format != "json" {
		fmt.Fprintln(errOut, "mycelium read: --format must be text or json")
		return ExitUsage
	}

	abs, err := resolveUnderMount(mount, requested)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium read: %v\n", err)
		if errors.Is(err, ErrMountUnset) {
			return ExitGenericError
		}
		return ExitUsage
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			fmt.Fprintf(errOut, "mycelium read: %s: not found\n", requested)
			return ExitGenericError
		}
		fmt.Fprintf(errOut, "mycelium read: %v\n", err)
		return ExitGenericError
	}

	if format == "json" {
		if !utf8.Valid(data) {
			fmt.Fprintf(errOut, "mycelium read: %s: --format json requires UTF-8 content\n", requested)
			return ExitGenericError
		}
		sum := sha256.Sum256(data)
		env := readEnvelope{
			Path:    requested,
			Version: versionPrefix + hex.EncodeToString(sum[:]),
			Content: string(data),
		}
		line, err := json.Marshal(env)
		if err != nil {
			fmt.Fprintf(errOut, "mycelium read: marshal: %v\n", err)
			return ExitGenericError
		}
		line = append(line, '\n')
		if _, err := out.Write(line); err != nil {
			fmt.Fprintf(errOut, "mycelium read: %v\n", err)
			return ExitGenericError
		}
		return ExitOK
	}

	if _, err := out.Write(data); err != nil {
		fmt.Fprintf(errOut, "mycelium read: %v\n", err)
		return ExitGenericError
	}
	return ExitOK
}

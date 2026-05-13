package mycelium

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
)

// scanJSONLinesFile calls fn for each non-empty JSONL line in filePath.
// It intentionally uses Reader.ReadBytes instead of bufio.Scanner so activity
// entries with large inline payloads are not rejected by Scanner's token limit.
func scanJSONLinesFile(filePath string, fn func([]byte) error) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	for {
		line, err := r.ReadBytes('\n')
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) > 0 {
			if cbErr := fn(trimmed); cbErr != nil {
				return cbErr
			}
		}
		if err == nil {
			continue
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
}

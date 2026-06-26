package mycelium

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
)

func legacyPendingTxDir(mount string) string {
	return filepath.Join(mount, "_tx", "pending")
}

func legacyPendingTxFiles(mount string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(legacyPendingTxDir(mount), "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	return matches, nil
}

func blockLegacyPendingTransactions(errOut io.Writer, mount string) int {
	matches, err := legacyPendingTxFiles(mount)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium: scan legacy pending transactions: %v\n", err)
		return ExitGenericError
	}
	if len(matches) == 0 {
		return ExitOK
	}

	first := relForwardSlash(mount, matches[0])
	fmt.Fprintf(
		errOut,
		"mycelium: legacy _tx/pending records found; current versions no longer replay the transaction journal. Run the last v0.2 mycelium binary on this mount to recover pending records, then retry. First pending record: %s\n",
		first,
	)
	return ExitGenericError
}

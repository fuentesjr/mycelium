package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"time"
)

type subcommand struct {
	name string
	run  func(in io.Reader, out, errOut io.Writer, args []string) int
}

var subcommands = []subcommand{
	{"read", runRead},
	{"write", runWrite},
	{"edit", runEdit},
	{"ls", runLs},
	{"glob", runGlob},
	{"grep", runGrep},
	{"rm", runRm},
	{"mv", runMv},
	{"log", runLog},
}

func dispatch(in io.Reader, out, errOut io.Writer, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(errOut, "usage: mycelium <subcommand> [args...]")
		printSubcommandList(errOut)
		return ExitUsage
	}
	name := args[0]
	rest := args[1:]
	for _, sc := range subcommands {
		if sc.name == name {
			return sc.run(in, out, errOut, rest)
		}
	}
	fmt.Fprintf(errOut, "mycelium: unknown subcommand: %s\n", name)
	printSubcommandList(errOut)
	return ExitUsage
}

func printSubcommandList(w io.Writer) {
	fmt.Fprint(w, "subcommands:")
	for _, sc := range subcommands {
		fmt.Fprintf(w, " %s", sc.name)
	}
	fmt.Fprintln(w)
}

func runRead(_ io.Reader, out, errOut io.Writer, args []string) int {
	fs := flag.NewFlagSet("read", flag.ContinueOnError)
	fs.SetOutput(errOut)
	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(positional) < 1 {
		fmt.Fprintln(errOut, "mycelium read: PATH required")
		return ExitUsage
	}
	return readFile(out, errOut, ReadIdentity().Mount, positional[0])
}

func runWrite(in io.Reader, out, errOut io.Writer, args []string) int {
	fs := flag.NewFlagSet("write", flag.ContinueOnError)
	fs.SetOutput(errOut)
	expectedVersion := fs.String("expected-version", "", "current version token for CAS")
	includeContent := fs.Bool("include-current-content", false, "include current file content in conflict envelope")
	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(positional) < 1 {
		fmt.Fprintln(errOut, "mycelium write: PATH required")
		return ExitUsage
	}
	id := ReadIdentity()
	// Check _-prefix reservation before calling writeFile.
	if _, resErr := resolveAgentWritable(id.Mount, positional[0]); resErr != nil {
		if errors.Is(resErr, ErrReservedPath) {
			fmt.Fprintf(errOut, "mycelium write: %s: writes to '_'-prefixed paths are reserved\n", positional[0])
			return ExitReservedPrefix
		}
		// Other path errors are handled inside writeFile; fall through.
	}
	version, rc := writeFile(in, errOut, id.Mount, positional[0], *expectedVersion, *includeContent)
	if rc != ExitOK {
		return rc
	}
	logMutation(errOut, id, MutationLog{Op: "write", Path: positional[0], Version: version})
	fmt.Fprintf(out, `{"version":%q}`+"\n", version)
	return ExitOK
}

func runEdit(_ io.Reader, out, errOut io.Writer, args []string) int {
	fs := flag.NewFlagSet("edit", flag.ContinueOnError)
	fs.SetOutput(errOut)
	expectedVersion := fs.String("expected-version", "", "current version token for CAS")
	includeContent := fs.Bool("include-current-content", false, "include current file content in conflict envelope")
	oldStr := fs.String("old", "", "string to replace")
	newStr := fs.String("new", "", "replacement string")
	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(positional) < 1 {
		fmt.Fprintln(errOut, "mycelium edit: PATH required")
		return ExitUsage
	}
	if *oldStr == "" {
		fmt.Fprintln(errOut, "mycelium edit: --old is required")
		return ExitUsage
	}
	id := ReadIdentity()
	// Check _-prefix reservation before calling editFile.
	if _, resErr := resolveAgentWritable(id.Mount, positional[0]); resErr != nil {
		if errors.Is(resErr, ErrReservedPath) {
			fmt.Fprintf(errOut, "mycelium edit: %s: writes to '_'-prefixed paths are reserved\n", positional[0])
			return ExitReservedPrefix
		}
	}
	version, rc := editFile(errOut, id.Mount, positional[0], *oldStr, *newStr, *expectedVersion, *includeContent)
	if rc != ExitOK {
		return rc
	}
	logMutation(errOut, id, MutationLog{Op: "edit", Path: positional[0], Version: version})
	fmt.Fprintf(out, `{"version":%q}`+"\n", version)
	return ExitOK
}

func runLs(_ io.Reader, out, errOut io.Writer, args []string) int {
	fs := flag.NewFlagSet("ls", flag.ContinueOnError)
	fs.SetOutput(errOut)
	recursive := fs.Bool("recursive", false, "recurse into subdirectories")
	if _, err := parseInterspersed(fs, args); err != nil {
		return ExitUsage
	}
	return listFilesAndPrint(out, errOut, ReadIdentity().Mount, *recursive)
}

func runGlob(_ io.Reader, out, errOut io.Writer, args []string) int {
	fs := flag.NewFlagSet("glob", flag.ContinueOnError)
	fs.SetOutput(errOut)
	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(positional) < 1 {
		fmt.Fprintln(errOut, "mycelium glob: PATTERN required")
		return ExitUsage
	}
	return globAndPrint(out, errOut, ReadIdentity().Mount, positional[0])
}

func runGrep(_ io.Reader, out, errOut io.Writer, args []string) int {
	fs := flag.NewFlagSet("grep", flag.ContinueOnError)
	fs.SetOutput(errOut)
	pattern := fs.String("pattern", "", "search pattern")
	pathScope := fs.String("path", "", "path to search under")
	useRegex := fs.Bool("regex", false, "treat pattern as regex")
	fileType := fs.String("file-type", "", "limit by file type")
	format := fs.String("format", "text", "output format: text|json")
	limit := fs.Int("limit", 1000, "max matches")
	if _, err := parseInterspersed(fs, args); err != nil {
		return ExitUsage
	}

	if *pattern == "" {
		fmt.Fprintln(errOut, "mycelium grep: --pattern is required")
		return ExitUsage
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintln(errOut, "mycelium grep: --format must be text or json")
		return ExitUsage
	}
	if *limit <= 0 {
		fmt.Fprintln(errOut, "mycelium grep: --limit must be positive")
		return ExitUsage
	}

	opts := GrepOptions{
		Pattern:   *pattern,
		PathScope: *pathScope,
		Regex:     *useRegex,
		FileType:  *fileType,
		Format:    *format,
		Limit:     *limit,
	}
	return grepFiles(out, errOut, ReadIdentity().Mount, opts)
}

func runRm(_ io.Reader, out, errOut io.Writer, args []string) int {
	fs := flag.NewFlagSet("rm", flag.ContinueOnError)
	fs.SetOutput(errOut)
	expectedVersion := fs.String("expected-version", "", "current version token for CAS")
	includeContent := fs.Bool("include-current-content", false, "include current file content in conflict envelope")
	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(positional) < 1 {
		fmt.Fprintln(errOut, "mycelium rm: PATH required")
		return ExitUsage
	}
	id := ReadIdentity()
	// Check _-prefix reservation before calling removeFile.
	if _, resErr := resolveAgentWritable(id.Mount, positional[0]); resErr != nil {
		if errors.Is(resErr, ErrReservedPath) {
			fmt.Fprintf(errOut, "mycelium rm: %s: writes to '_'-prefixed paths are reserved\n", positional[0])
			return ExitReservedPrefix
		}
	}
	priorVersion, rc := removeFile(errOut, id.Mount, positional[0], *expectedVersion, *includeContent)
	if rc != ExitOK {
		return rc
	}
	logMutation(errOut, id, MutationLog{Op: "rm", Path: positional[0], PriorVersion: priorVersion})
	return ExitOK
}

func runMv(_ io.Reader, out, errOut io.Writer, args []string) int {
	fs := flag.NewFlagSet("mv", flag.ContinueOnError)
	fs.SetOutput(errOut)
	expectedVersion := fs.String("expected-version", "", "current version token for CAS")
	includeContent := fs.Bool("include-current-content", false, "include current file content in conflict envelope")
	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(positional) < 2 {
		fmt.Fprintln(errOut, "mycelium mv: SRC and DST required")
		return ExitUsage
	}
	id := ReadIdentity()
	src, dst := positional[0], positional[1]
	// Check _-prefix reservation for both src and dst.
	if _, resErr := resolveAgentWritable(id.Mount, src); resErr != nil {
		if errors.Is(resErr, ErrReservedPath) {
			fmt.Fprintf(errOut, "mycelium mv: %s: writes to '_'-prefixed paths are reserved\n", src)
			return ExitReservedPrefix
		}
	}
	if _, resErr := resolveAgentWritable(id.Mount, dst); resErr != nil {
		if errors.Is(resErr, ErrReservedPath) {
			fmt.Fprintf(errOut, "mycelium mv: %s: writes to '_'-prefixed paths are reserved\n", dst)
			return ExitReservedPrefix
		}
	}
	version, rc := moveFile(errOut, id.Mount, src, dst, *expectedVersion, *includeContent)
	if rc != ExitOK {
		return rc
	}
	logMutation(errOut, id, MutationLog{Op: "mv", Path: dst, From: src, Version: version})
	return ExitOK
}

func runLog(in io.Reader, _ io.Writer, errOut io.Writer, args []string) int {
	fs := flag.NewFlagSet("log", flag.ContinueOnError)
	fs.SetOutput(errOut)
	pathFlag := fs.String("path", "", "path to record on the entry")
	payloadJSON := fs.String("payload-json", "", "inline JSON payload")
	fromStdin := fs.Bool("stdin", false, "read payload from stdin")
	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(positional) < 1 {
		fmt.Fprintln(errOut, "mycelium log: OP required")
		return ExitUsage
	}
	if *payloadJSON != "" && *fromStdin {
		fmt.Fprintln(errOut, "mycelium log: --payload-json and --stdin are mutually exclusive")
		return ExitUsage
	}
	op := positional[0]
	return appendLog(in, errOut, ReadIdentity(), op, *pathFlag, *payloadJSON, *fromStdin, time.Now())
}

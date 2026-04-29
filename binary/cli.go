package main

import (
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
	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(positional) < 1 {
		fmt.Fprintln(errOut, "mycelium write: PATH required")
		return ExitUsage
	}
	return writeFile(in, out, errOut, ReadIdentity().Mount, positional[0], *expectedVersion)
}

func runEdit(_ io.Reader, out, errOut io.Writer, args []string) int {
	fs := flag.NewFlagSet("edit", flag.ContinueOnError)
	fs.SetOutput(errOut)
	fs.String("expected-version", "", "current version token for CAS")
	fs.String("old", "", "string to replace")
	fs.String("new", "", "replacement string")
	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(positional) < 1 {
		fmt.Fprintln(errOut, "mycelium edit: PATH required")
		return ExitUsage
	}
	return stubWriteOrEdit(out)
}

func runLs(_ io.Reader, out, errOut io.Writer, args []string) int {
	fs := flag.NewFlagSet("ls", flag.ContinueOnError)
	fs.SetOutput(errOut)
	fs.Bool("recursive", false, "recurse into subdirectories")
	if _, err := parseInterspersed(fs, args); err != nil {
		return ExitUsage
	}
	return stubListing(out)
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
	return stubListing(out)
}

func runGrep(_ io.Reader, _, errOut io.Writer, args []string) int {
	fs := flag.NewFlagSet("grep", flag.ContinueOnError)
	fs.SetOutput(errOut)
	fs.String("pattern", "", "search pattern")
	fs.String("path", "", "path to search under")
	fs.Bool("regex", false, "treat pattern as regex")
	fs.String("file-type", "", "limit by file type")
	fs.String("format", "text", "output format: text|json")
	fs.Int("limit", 1000, "max matches")
	fs.String("cursor", "", "pagination cursor")
	if _, err := parseInterspersed(fs, args); err != nil {
		return ExitUsage
	}
	return stubGrep()
}

func runRm(_ io.Reader, out, errOut io.Writer, args []string) int {
	fs := flag.NewFlagSet("rm", flag.ContinueOnError)
	fs.SetOutput(errOut)
	fs.String("expected-version", "", "current version token for CAS")
	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(positional) < 1 {
		fmt.Fprintln(errOut, "mycelium rm: PATH required")
		return ExitUsage
	}
	return stubLogOrRemove(out)
}

func runMv(_ io.Reader, out, errOut io.Writer, args []string) int {
	fs := flag.NewFlagSet("mv", flag.ContinueOnError)
	fs.SetOutput(errOut)
	fs.String("expected-version", "", "current version token for CAS")
	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(positional) < 2 {
		fmt.Fprintln(errOut, "mycelium mv: SRC and DST required")
		return ExitUsage
	}
	return stubLogOrRemove(out)
}

func runLog(in io.Reader, out, errOut io.Writer, args []string) int {
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
	return appendLog(in, out, errOut, ReadIdentity(), op, *pathFlag, *payloadJSON, *fromStdin, time.Now())
}

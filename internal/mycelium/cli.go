package mycelium

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
	{"grep", runGrep},
	{"rm", runRm},
	{"mv", runMv},
	{"log", runLog},
	{"evolve", runEvolve},
}

// Dispatch runs the mycelium CLI with injected streams and arguments.
func Dispatch(in io.Reader, out, errOut io.Writer, args []string) int {
	return dispatch(in, out, errOut, args)
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
	fmt.Fprintln(w, "subcommands:")
	fmt.Fprintln(w, "  everyday:   read write edit ls grep")
	fmt.Fprintln(w, "  occasional: rm mv")
	fmt.Fprintln(w, "  metadata:   log")
}

func requirePositionalArgs(errOut io.Writer, command string, positional []string, want int, required string) int {
	if len(positional) == want {
		return ExitOK
	}
	if len(positional) < want {
		fmt.Fprintf(errOut, "mycelium %s: %s required\n", command, required)
		return ExitUsage
	}
	fmt.Fprintf(errOut, "mycelium %s: unexpected argument: %s\n", command, positional[want])
	return ExitUsage
}

func runRead(_ io.Reader, out, errOut io.Writer, args []string) int {
	fs := flag.NewFlagSet("read", flag.ContinueOnError)
	fs.SetOutput(errOut)
	format := fs.String("format", "text", "output format: text|json")
	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if rc := requirePositionalArgs(errOut, "read", positional, 1, "PATH"); rc != ExitOK {
		return rc
	}
	return readFile(out, errOut, ReadIdentity().Mount, positional[0], *format)
}

func runWrite(in io.Reader, out, errOut io.Writer, args []string) int {
	fs := flag.NewFlagSet("write", flag.ContinueOnError)
	fs.SetOutput(errOut)
	expectedVersion := fs.String("expected-version", "", "current version token for CAS")
	rationaleFlag := fs.String("rationale", "", "optional reasoning captured to the activity log (≤64 KiB)")
	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if rc := requirePositionalArgs(errOut, "write", positional, 1, "PATH"); rc != ExitOK {
		return rc
	}
	if len(*rationaleFlag) > maxRationaleSize {
		fmt.Fprintf(errOut, "mycelium write: --rationale exceeds %d bytes\n", maxRationaleSize)
		return ExitReservedPrefix
	}
	id := ReadIdentity()
	// Check _-prefix reservation before reading stdin and entering the locked mutation.
	if _, resErr := resolveAgentWritable(id.Mount, positional[0]); resErr != nil {
		if errors.Is(resErr, ErrReservedPath) {
			fmt.Fprintf(errOut, "mycelium write: %s: writes to '_'-prefixed paths are reserved\n", positional[0])
			return ExitReservedPrefix
		}
		// Other path errors are handled inside mutatingWrite; fall through.
	}
	content, err := io.ReadAll(in)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium write: read stdin: %v\n", err)
		return ExitGenericError
	}
	version, rc := mutatingWrite(errOut, id, positional[0], content, *expectedVersion, *rationaleFlag)
	if rc != ExitOK {
		return rc
	}
	fmt.Fprintf(out, `{"version":%q}`+"\n", version)
	return ExitOK
}

func runEdit(_ io.Reader, out, errOut io.Writer, args []string) int {
	fs := flag.NewFlagSet("edit", flag.ContinueOnError)
	fs.SetOutput(errOut)
	expectedVersion := fs.String("expected-version", "", "current version token for CAS")
	oldStr := fs.String("old", "", "string to replace")
	newStr := fs.String("new", "", "replacement string")
	rationaleFlag := fs.String("rationale", "", "optional reasoning captured to the activity log (≤64 KiB)")
	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if rc := requirePositionalArgs(errOut, "edit", positional, 1, "PATH"); rc != ExitOK {
		return rc
	}
	if *oldStr == "" {
		fmt.Fprintln(errOut, "mycelium edit: --old is required")
		return ExitUsage
	}
	if len(*rationaleFlag) > maxRationaleSize {
		fmt.Fprintf(errOut, "mycelium edit: --rationale exceeds %d bytes\n", maxRationaleSize)
		return ExitReservedPrefix
	}
	id := ReadIdentity()
	// Check _-prefix reservation before entering the locked mutation.
	if _, resErr := resolveAgentWritable(id.Mount, positional[0]); resErr != nil {
		if errors.Is(resErr, ErrReservedPath) {
			fmt.Fprintf(errOut, "mycelium edit: %s: writes to '_'-prefixed paths are reserved\n", positional[0])
			return ExitReservedPrefix
		}
	}
	version, rc := mutatingEdit(errOut, id, positional[0], *oldStr, *newStr, *expectedVersion, *rationaleFlag)
	if rc != ExitOK {
		return rc
	}
	fmt.Fprintf(out, `{"version":%q}`+"\n", version)
	return ExitOK
}

func runLs(_ io.Reader, out, errOut io.Writer, args []string) int {
	fs := flag.NewFlagSet("ls", flag.ContinueOnError)
	fs.SetOutput(errOut)
	recursive := fs.Bool("recursive", false, "recurse into subdirectories")
	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(positional) > 1 {
		fmt.Fprintf(errOut, "mycelium ls: unexpected argument: %s\n", positional[1])
		return ExitUsage
	}
	pattern := ""
	if len(positional) == 1 {
		pattern = positional[0]
	}
	return listFilesAndPrint(out, errOut, ReadIdentity().Mount, *recursive, pattern)
}

func runGrep(_ io.Reader, out, errOut io.Writer, args []string) int {
	fs := flag.NewFlagSet("grep", flag.ContinueOnError)
	fs.SetOutput(errOut)
	pattern := fs.String("pattern", "", "search pattern")
	pathScope := fs.String("path", "", "path to search under")
	useRegex := fs.Bool("regex", false, "treat pattern as regex")
	format := fs.String("format", "text", "output format: text|json")
	limit := fs.Int("limit", 1000, "max matches")
	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if rc := requirePositionalArgs(errOut, "grep", positional, 0, ""); rc != ExitOK {
		return rc
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
		Format:    *format,
		Limit:     *limit,
	}
	return grepFiles(out, errOut, ReadIdentity().Mount, opts)
}

func runRm(_ io.Reader, out, errOut io.Writer, args []string) int {
	fs := flag.NewFlagSet("rm", flag.ContinueOnError)
	fs.SetOutput(errOut)
	expectedVersion := fs.String("expected-version", "", "current version token for CAS")
	rationaleFlag := fs.String("rationale", "", "optional reasoning captured to the activity log (≤64 KiB)")
	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if rc := requirePositionalArgs(errOut, "rm", positional, 1, "PATH"); rc != ExitOK {
		return rc
	}
	if len(*rationaleFlag) > maxRationaleSize {
		fmt.Fprintf(errOut, "mycelium rm: --rationale exceeds %d bytes\n", maxRationaleSize)
		return ExitReservedPrefix
	}
	id := ReadIdentity()
	// Check _-prefix reservation before entering the locked mutation.
	if _, resErr := resolveAgentWritable(id.Mount, positional[0]); resErr != nil {
		if errors.Is(resErr, ErrReservedPath) {
			fmt.Fprintf(errOut, "mycelium rm: %s: writes to '_'-prefixed paths are reserved\n", positional[0])
			return ExitReservedPrefix
		}
	}
	_, rc := mutatingRemove(errOut, id, positional[0], *expectedVersion, *rationaleFlag)
	if rc != ExitOK {
		return rc
	}
	return ExitOK
}

func runMv(_ io.Reader, out, errOut io.Writer, args []string) int {
	fs := flag.NewFlagSet("mv", flag.ContinueOnError)
	fs.SetOutput(errOut)
	expectedVersion := fs.String("expected-version", "", "current version token for CAS")
	rationaleFlag := fs.String("rationale", "", "optional reasoning captured to the activity log (≤64 KiB)")
	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if rc := requirePositionalArgs(errOut, "mv", positional, 2, "SRC and DST"); rc != ExitOK {
		return rc
	}
	if len(*rationaleFlag) > maxRationaleSize {
		fmt.Fprintf(errOut, "mycelium mv: --rationale exceeds %d bytes\n", maxRationaleSize)
		return ExitReservedPrefix
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
	_, rc := mutatingMove(errOut, id, src, dst, *expectedVersion, *rationaleFlag)
	if rc != ExitOK {
		return rc
	}
	return ExitOK
}

func runLog(in io.Reader, _ io.Writer, errOut io.Writer, args []string) int {
	fs := flag.NewFlagSet("log", flag.ContinueOnError)
	fs.SetOutput(errOut)
	pathFlag := fs.String("path", "", "path to record on the entry")
	payloadJSON := fs.String("payload-json", "", "inline JSON payload")
	fromStdin := fs.Bool("stdin", false, "read payload from stdin")
	rationaleFlag := fs.String("rationale", "", "optional reasoning captured to the activity log (≤64 KiB)")
	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if rc := requirePositionalArgs(errOut, "log", positional, 1, "OP"); rc != ExitOK {
		return rc
	}
	if *payloadJSON != "" && *fromStdin {
		fmt.Fprintln(errOut, "mycelium log: --payload-json and --stdin are mutually exclusive")
		return ExitUsage
	}
	if len(*rationaleFlag) > maxRationaleSize {
		fmt.Fprintf(errOut, "mycelium log: --rationale exceeds %d bytes\n", maxRationaleSize)
		return ExitReservedPrefix
	}
	op := positional[0]
	return appendLog(in, errOut, ReadIdentity(), op, *pathFlag, *payloadJSON, *fromStdin, *rationaleFlag, time.Now())
}

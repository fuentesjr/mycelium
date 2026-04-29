package main

import "flag"

// parseInterspersed parses args allowing flags to appear before, between, or
// after positional arguments. The Go stdlib flag package stops at the first
// non-flag token; this helper resumes parsing after each positional.
func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		if fs.NArg() == 0 {
			break
		}
		positional = append(positional, fs.Arg(0))
		args = fs.Args()[1:]
	}
	return positional, nil
}

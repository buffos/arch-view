package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/buffo/arch-view/internal/analysis"
	appversion "github.com/buffo/arch-view/internal/version"
)

func runVersionCommand(_ *analysis.Host, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("arch-view version", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "write machine-readable version metadata")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "version does not accept positional arguments", map[string]any{"arguments": fs.Args()})
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}

	info := appversion.Current()
	if *jsonOutput {
		return writeJSON(stdout, info)
	}
	if _, err := fmt.Fprintln(stdout, info.String()); err != nil {
		return 1
	}
	return 0
}

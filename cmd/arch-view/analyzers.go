package main

import (
	"flag"
	"io"

	"github.com/buffo/arch-view/internal/analysis"
)

func runAnalyzersCommand(base *analysis.Host, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("arch-view analyzers", flag.ContinueOnError)
	fs.SetOutput(stderr)
	analyzerRuntime := fs.String("analyzer-runtime", analyzerRuntimeAuto, "analyzer runtime: auto, packaged, in-process, or explicit")
	allowUntrustedPlugin := fs.Bool("allow-untrusted-plugin", false, "allow an explicitly supplied local descriptor")
	var plugins stringList
	fs.Var(&plugins, "plugin", "external analyzer descriptor; repeatable")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "analyzers does not accept positional arguments", nil)
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}

	host, catalog, err := configureCommandHost(base, commandRuntimeOptions{
		Mode:                 *analyzerRuntime,
		ModeProvided:         runtimeModeWasProvided(fs),
		AllowUntrustedPlugin: *allowUntrustedPlugin,
		PluginPaths:          []string(plugins),
		ListOnly:             true,
	})
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	listings := packageListings(host, catalog)
	sortAnalyzerListings(listings)
	return writeJSON(stdout, struct {
		Analyzers []analyzerListing `json:"analyzers"`
	}{Analyzers: listings})
}

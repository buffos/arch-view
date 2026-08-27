package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/goanalyzer"
	"github.com/buffo/arch-view/internal/pyanalyzer"
)

type stringList []string

func (s *stringList) String() string {
	return strings.Join(*s, ",")
}

func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	registry := analysis.NewRegistry()
	if err := registry.Register(goanalyzer.New()); err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if err := registry.Register(pyanalyzer.New()); err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	host := analysis.NewHost(registry)
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "analyzers":
		if len(args) != 1 {
			err := analysis.NewHostError(analysis.ErrInvalidRequest, "analyzers does not accept positional arguments", nil)
			writeError(stderr, err)
			return analysis.ExitCodeForError(err)
		}
		return writeJSON(stdout, struct {
			Analyzers []analysis.Manifest `json:"analyzers"`
		}{Analyzers: host.ListManifests()})
	case "analyze":
		return runAnalyze(host, args[1:], stdout, stderr)
	case "open":
		return runOpen(host, args[1:], stdout, stderr)
	case "model":
		return runModel(args[1:], stdout, stderr)
	case "export":
		return runExport(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	default:
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "unknown command", map[string]any{"command": args[0]})
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
}

func printUsage(writer io.Writer) {
	_, _ = fmt.Fprintln(writer, "arch-view analyzers")
	_, _ = fmt.Fprintln(writer, "arch-view analyze --project <path> [--language <id>] [--module <path>] [--source-root <path>] [--python-version <3.x>] [--include-stubs] --format analysis-json|json|html|svg --output <file>")
	_, _ = fmt.Fprintln(writer, "arch-view export --input <model.json> --format json|html|svg --output <file>")
	_, _ = fmt.Fprintln(writer, "arch-view open --model <model.json> [--port <n>]")
	_, _ = fmt.Fprintln(writer, "arch-view open --project <path> [--language <id>] [--module <path>] [--source-root <path>] [--python-version <3.x>] [--include-stubs] [--port <n>]")
	_, _ = fmt.Fprintln(writer, "arch-view model normalize --input <analysis-json> --output <model-json>")
	_, _ = fmt.Fprintln(writer, "arch-view model validate --input <model-json>")
	_, _ = fmt.Fprintln(writer, "arch-view model projection --input <model-json> [--path <segment>] --output <projection-json>")
}

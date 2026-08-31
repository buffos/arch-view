package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analyzers/clojure"
	"github.com/buffo/arch-view/internal/analyzers/go"
	"github.com/buffo/arch-view/internal/analyzers/python"
	"github.com/buffo/arch-view/internal/analyzers/rust"
	"github.com/buffo/arch-view/internal/analyzers/typescript"
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
	host, err := newBuiltInHost()
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "analyzers":
		return runAnalyzersCommand(host, args[1:], stdout, stderr)
	case "analyze":
		return runAnalyze(host, args[1:], stdout, stderr)
	case "quality":
		return runQualityCommand(host, args[1:], stdout, stderr)
	case "open":
		return runOpen(host, args[1:], stdout, stderr)
	case "live":
		return runLiveCommand(host, args[1:], stdout, stderr)
	case "mcp":
		return runMCPCommand(host, args[1:], stdout, stderr)
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

func newBuiltInHost() (*analysis.Host, error) {
	registry := analysis.NewRegistry()
	for _, analyzer := range []analysis.Analyzer{
		goanalyzer.New(),
		clojureanalyzer.New(),
		pyanalyzer.New(),
		rustanalyzer.New(),
		tsanalyzer.New(),
	} {
		if err := registry.Register(analyzer); err != nil {
			return nil, err
		}
	}
	return analysis.NewHost(registry), nil
}

func printUsage(writer io.Writer) {
	_, _ = fmt.Fprintln(writer, "arch-view analyzers [--analyzer-runtime auto|packaged|in-process|explicit] [--plugin <descriptor>] [--allow-untrusted-plugin]")
	_, _ = fmt.Fprintln(writer, "arch-view analyze --project <path> [--analyzer-runtime auto|packaged|in-process|explicit] [--plugin <descriptor>] [--allow-untrusted-plugin] [--language <id>] [--analyzer <id>] [--module <path>] [--crate <name-or-path>] [--feature <name>] [--target <triple>] [--config <tsconfig path>] [--source-root <path>] [--platform <clj|cljs|both>] [--python-version <3.x>] [--include-stubs] [--include-js] [--include-tests] [--include-examples] [--runtime auto|esm|cjs] [--exclude <glob>] [--quality-profile <profile.json>] [--quality-baseline <baseline.json>] [--quality-exit-on info|warning|error|blocker] --format analysis-json|json|html|svg --output <file>")
	_, _ = fmt.Fprintln(writer, "arch-view quality baseline --input <analysis|model|quality-report.json> --output <baseline.json|-> --baseline-id baseline:<name> (--finding <finding-id-or-key> ... | --all-active) --reason <text> [--owner <name>] [--revision <version>] [--overwrite]")
	_, _ = fmt.Fprintln(writer, "arch-view export --input <model.json> --format json|html|svg --output <file> [--quality-exit-on info|warning|error|blocker]")
	_, _ = fmt.Fprintln(writer, "arch-view open --model <model.json> [--port <n>]")
	_, _ = fmt.Fprintln(writer, "arch-view open --project <path> [--analyzer-runtime auto|packaged|in-process|explicit] [--plugin <descriptor>] [--allow-untrusted-plugin] [--language <id>] [--analyzer <id>] [--module <path>] [--crate <name-or-path>] [--feature <name>] [--target <triple>] [--config <tsconfig path>] [--source-root <path>] [--platform <clj|cljs|both>] [--python-version <3.x>] [--include-stubs] [--include-js] [--include-tests] [--include-examples] [--runtime auto|esm|cjs] [--exclude <glob>] [--port <n>]")
	_, _ = fmt.Fprintln(writer, "arch-view live start|status|wait|ensure-current ...")
	_, _ = fmt.Fprintln(writer, "arch-view mcp --project <path> [--transport stdio|local_http|authenticated_http]")
	_, _ = fmt.Fprintln(writer, "arch-view model normalize --input <analysis-json> --output <model-json>")
	_, _ = fmt.Fprintln(writer, "arch-view model validate --input <model-json>")
	_, _ = fmt.Fprintln(writer, "arch-view model projection --input <model-json> [--path <segment>] --output <projection-json>")
}

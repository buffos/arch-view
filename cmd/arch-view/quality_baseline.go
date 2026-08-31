package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/quality"
)

func runQualityCommand(_ *analysis.Host, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printQualityUsage(stderr)
		return 2
	}
	switch args[0] {
	case "baseline":
		if len(args) > 1 && strings.EqualFold(args[1], "add") {
			return runQualityBaselineAdd(args[2:], stdout, stderr)
		}
		return runQualityBaseline(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		printQualityUsage(stdout)
		return 0
	default:
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "unknown quality command", map[string]any{"command": args[0]})
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
}

func runQualityBaseline(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("arch-view quality baseline", flag.ContinueOnError)
	fs.SetOutput(stderr)
	input := fs.String("input", "", "analysis, model, or quality-report JSON file")
	output := fs.String("output", "", "baseline JSON file, or - for stdout")
	baselineID := fs.String("baseline-id", "", "namespaced baseline identity, for example baseline:main")
	revision := fs.String("revision", "", "optional baseline revision")
	reason := fs.String("reason", "", "reason recorded for each baseline entry")
	owner := fs.String("owner", "", "optional owner recorded for each baseline entry")
	allActive := fs.Bool("all-active", false, "baseline every active finding in the report")
	overwrite := fs.Bool("overwrite", false, "replace an existing baseline file")
	var findings stringList
	fs.Var(&findings, "finding", "finding ID or stable finding key; repeatable")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			printQualityUsage(stdout)
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "quality baseline does not accept positional arguments", map[string]any{"arguments": fs.Args()})
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if strings.TrimSpace(*input) == "" || strings.TrimSpace(*output) == "" || strings.TrimSpace(*baselineID) == "" || strings.TrimSpace(*reason) == "" {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "quality baseline requires --input, --output, --baseline-id, and --reason", nil)
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if *allActive && len(findings) > 0 {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "--all-active cannot be combined with --finding", nil)
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	report, err := readQualityReport(*input)
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	selected := append([]string(nil), findings...)
	if *allActive {
		for _, finding := range report.Findings {
			if finding.Status == quality.StatusActive {
				selected = append(selected, finding.ID)
			}
		}
	}
	if len(selected) == 0 {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "quality baseline requires at least one --finding or --all-active", nil)
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}

	baseline := quality.Baseline{
		SchemaVersion: quality.BaselineSchemaVersion,
		BaselineID:    strings.TrimSpace(*baselineID),
		Revision:      strings.TrimSpace(*revision),
		Entries:       []quality.BaselineEntry{},
		Extensions:    []quality.ExtensionBlock{},
	}
	for _, findingID := range selected {
		entry, entryErr := quality.CreateBaselineEntry(report, findingID, *reason, *owner)
		if entryErr != nil {
			err := analysis.WrapHostError(analysis.ErrInvalidOptions, "quality baseline entry could not be created", entryErr, map[string]any{"finding_id": findingID})
			writeError(stderr, err)
			return analysis.ExitCodeForError(err)
		}
		baseline, entryErr = quality.AddBaselineEntry(baseline, entry)
		if entryErr != nil {
			err := analysis.WrapHostError(analysis.ErrInvalidOptions, "quality baseline entry could not be added", entryErr, map[string]any{"finding_id": findingID})
			writeError(stderr, err)
			return analysis.ExitCodeForError(err)
		}
	}
	if err := quality.ValidateBaseline(baseline); err != nil {
		err = analysis.WrapHostError(analysis.ErrInvalidOptions, "generated quality baseline is invalid", err, nil)
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if *output == "-" {
		return writeJSON(stdout, baseline)
	}
	if err := writeQualityBaselineFile(*output, baseline, *overwrite); err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	_, _ = fmt.Fprintf(stdout, "Quality baseline written to %s (%d entries).\n", *output, len(baseline.Entries))
	return 0
}

func readQualityReport(path string) (quality.QualityEvaluation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return quality.QualityEvaluation{}, analysis.WrapHostError(analysis.ErrInvalidOptions, "quality report input could not be read", err, map[string]any{"input": path})
	}
	var envelope struct {
		QualityReport *quality.QualityEvaluation `json:"quality_report"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return quality.QualityEvaluation{}, analysis.WrapHostError(analysis.ErrInvalidOptions, "quality report input JSON is invalid", err, map[string]any{"input": path})
	}
	if envelope.QualityReport != nil {
		if err := quality.ValidateQualityEvaluation(*envelope.QualityReport); err != nil {
			return quality.QualityEvaluation{}, analysis.WrapHostError(analysis.ErrInvalidModel, "quality report input is invalid", err, map[string]any{"input": path})
		}
		return *envelope.QualityReport, nil
	}
	var report quality.QualityEvaluation
	if err := json.Unmarshal(data, &report); err != nil || report.SchemaVersion == "" {
		return quality.QualityEvaluation{}, analysis.NewHostError(analysis.ErrInvalidModel, "input does not contain a quality report", map[string]any{"input": path})
	}
	if err := quality.ValidateQualityEvaluation(report); err != nil {
		return quality.QualityEvaluation{}, analysis.WrapHostError(analysis.ErrInvalidModel, "quality report input is invalid", err, map[string]any{"input": path})
	}
	return report, nil
}

func writeQualityBaselineFile(path string, baseline quality.Baseline, overwrite bool) error {
	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return analysis.WrapHostError(analysis.ErrHostFailure, "quality baseline could not be encoded", err, map[string]any{"output": path})
	}
	data = append(data, '\n')
	if info, statErr := os.Lstat(path); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return analysis.NewHostError(analysis.ErrInvalidRequest, "quality baseline destination may not be a symlink", map[string]any{"output": path})
		}
		if !overwrite {
			return analysis.NewHostError(analysis.ErrInvalidRequest, "quality baseline destination already exists; pass --overwrite to replace it", map[string]any{"output": path})
		}
	} else if !os.IsNotExist(statErr) {
		return analysis.WrapHostError(analysis.ErrHostFailure, "quality baseline destination could not be inspected", statErr, map[string]any{"output": path})
	}
	flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if !overwrite {
		flags = os.O_WRONLY | os.O_CREATE | os.O_EXCL
	}
	file, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		if !overwrite && errors.Is(err, os.ErrExist) {
			return analysis.NewHostError(analysis.ErrInvalidRequest, "quality baseline destination already exists; pass --overwrite to replace it", map[string]any{"output": path})
		}
		return analysis.WrapHostError(analysis.ErrHostFailure, "quality baseline could not be written", err, map[string]any{"output": path})
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return analysis.WrapHostError(analysis.ErrHostFailure, "quality baseline could not be written", err, map[string]any{"output": path})
	}
	if err := file.Close(); err != nil {
		return analysis.WrapHostError(analysis.ErrHostFailure, "quality baseline could not be closed", err, map[string]any{"output": path})
	}
	return nil
}

func printQualityUsage(writer io.Writer) {
	_, _ = fmt.Fprintln(writer, "arch-view quality baseline --input <analysis|model|quality-report.json> --output <baseline.json|-> --baseline-id baseline:<name> --finding <finding-id-or-key> [--finding <...>] --reason <text> [--owner <name>] [--revision <version>] [--overwrite]")
	_, _ = fmt.Fprintln(writer, "arch-view quality baseline --input <analysis|model|quality-report.json> --output <baseline.json|-> --baseline-id baseline:<name> --all-active --reason <text> [--owner <name>] [--revision <version>] [--overwrite]")
	_, _ = fmt.Fprintln(writer, "arch-view quality baseline add --project <path> --profile <quality-profiles/profile.json> --baseline-file <name.json> --input <analysis|quality-report.json> (--finding <finding-id-or-key> ... | --all-active) --reason <text> [--owner <name>] [--revision <version>] [--expected-revision <version>]")
}

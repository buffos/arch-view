package main

import (
	"encoding/json"
	"flag"
	"io"
	"os"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/model/canonical"
)

func runModel(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "model requires a subcommand", nil)
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	switch args[0] {
	case "normalize":
		return runModelNormalize(args[1:], stdout, stderr)
	case "validate":
		return runModelValidate(args[1:], stdout, stderr)
	case "projection":
		return runModelProjection(args[1:], stdout, stderr)
	default:
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "unknown model subcommand", map[string]any{"command": args[0]})
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
}

func runModelNormalize(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("arch-view model normalize", flag.ContinueOnError)
	fs.SetOutput(stderr)
	input := fs.String("input", "", "analysis JSON input file")
	output := fs.String("output", "", "model JSON output file, or - for stdout")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "model normalize does not accept positional arguments", map[string]any{"arguments": fs.Args()})
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if *input == "" || *output == "" {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "model normalize requires --input and --output", nil)
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	data, err := os.ReadFile(*input)
	if err != nil {
		hostErr := analysis.WrapHostError(analysis.ErrInvalidModel, "analysis JSON input could not be read", err, map[string]any{"input": *input})
		writeError(stderr, hostErr)
		return analysis.ExitCodeForError(hostErr)
	}
	var result analysis.AnalysisResult
	if err := json.Unmarshal(data, &result); err != nil {
		hostErr := analysis.WrapHostError(analysis.ErrInvalidModel, "analysis JSON input is invalid", err, map[string]any{"input": *input})
		writeError(stderr, hostErr)
		return analysis.ExitCodeForError(hostErr)
	}
	normalized, err := canonical.Normalize(result)
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if *output == "-" {
		return writeJSON(stdout, normalized)
	}
	if err := writeFileJSON(*output, normalized); err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	return analysis.ExitCodeForStatus(analysis.AnalysisStatus(normalized.Status))
}

func runModelValidate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("arch-view model validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	input := fs.String("input", "", "model JSON input file")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "model validate does not accept positional arguments", map[string]any{"arguments": fs.Args()})
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if *input == "" {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "model validate requires --input", nil)
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	value, err := readModelFile(*input)
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if err := canonical.Validate(value); err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	return writeJSON(stdout, struct {
		Valid   bool   `json:"valid"`
		ModelID string `json:"model_id"`
	}{Valid: true, ModelID: value.ModelID})
}

func runModelProjection(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("arch-view model projection", flag.ContinueOnError)
	fs.SetOutput(stderr)
	input := fs.String("input", "", "model JSON input file")
	output := fs.String("output", "", "projection JSON output file, or - for stdout")
	var selectedPath stringList
	fs.Var(&selectedPath, "path", "hierarchy segment; repeatable")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "model projection does not accept positional arguments", map[string]any{"arguments": fs.Args()})
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if *input == "" || *output == "" {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "model projection requires --input and --output", nil)
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	value, err := readModelFile(*input)
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if err := canonical.Validate(value); err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	projection, err := model.BuildHierarchyProjection(value, []string(selectedPath))
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if *output == "-" {
		return writeJSON(stdout, projection)
	}
	if err := writeFileJSON(*output, projection); err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	return 0
}

func readModelFile(path string) (model.Model, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.Model{}, analysis.WrapHostError(analysis.ErrInvalidModel, "model JSON input could not be read", err, map[string]any{"input": path})
	}
	var value model.Model
	if err := json.Unmarshal(data, &value); err != nil {
		return model.Model{}, analysis.WrapHostError(analysis.ErrInvalidModel, "model JSON input is invalid", err, map[string]any{"input": path})
	}
	return value, nil
}

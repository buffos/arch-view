package main

import (
	"context"
	"flag"
	"io"
	"os"
	"os/signal"

	"github.com/buffo/arch-view/internal/analysis"
	exporter "github.com/buffo/arch-view/internal/export"
	"github.com/buffo/arch-view/internal/model"
)

func runExport(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("arch-view export", flag.ContinueOnError)
	fs.SetOutput(stderr)
	input := fs.String("input", "", "canonical model JSON input file")
	format := fs.String("format", "", "export format: json, html, or svg")
	output := fs.String("output", "", "export output file")
	referenceVisibility := fs.String("reference-visibility", "hidden", "visual reference visibility: hidden, aggregated, or expanded")
	deterministic := fs.Bool("deterministic", true, "produce deterministic output")
	overwrite := fs.Bool("overwrite", false, "replace an existing export output")
	embedSource := fs.Bool("embed-source", false, "embed source contents; unsupported in v1")
	var viewPath stringList
	var referenceScopes stringList
	fs.Var(&viewPath, "view-path", "hierarchy segment for visual export; repeatable")
	fs.Var(&referenceScopes, "reference-scope", "reference scope for visual export; repeatable")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "export does not accept positional arguments", map[string]any{"arguments": fs.Args()})
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if *input == "" || *format == "" || *output == "" {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "export requires --input, --format, and --output", nil)
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if !*deterministic {
		err := analysis.NewHostError(analysis.ErrUnsupportedOption, "non-deterministic export is unsupported", map[string]any{"deterministic": false})
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	value, err := readModelFile(*input)
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return writeExport(value, exporter.Request{Format: *format, OutputPath: *output, ViewPath: []string(viewPath), ReferenceVisibility: *referenceVisibility, ReferenceScopes: []string(referenceScopes), Overwrite: *overwrite, EmbedSource: *embedSource, Context: ctx}, stdout, stderr)
}

func writeExport(value model.Model, request exporter.Request, stdout, stderr io.Writer) int {
	metadata, err := exporter.Write(value, request)
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if code := writeJSON(stdout, metadata); code != 0 {
		return code
	}
	return analysis.ExitCodeForStatus(analysis.AnalysisStatus(metadata.Status))
}

func supportedExportFormat(value string) bool {
	return value == exporter.FormatJSON || value == exporter.FormatHTML || value == exporter.FormatSVG
}

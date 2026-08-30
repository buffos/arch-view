// Command docs-inventory emits the machine-readable product surface used by
// the public documentation build. It is a developer tool, not an Arch View
// runtime command.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	clojureanalyzer "github.com/buffo/arch-view/internal/analyzers/clojure"
	goanalyzer "github.com/buffo/arch-view/internal/analyzers/go"
	pyanalyzer "github.com/buffo/arch-view/internal/analyzers/python"
	rustanalyzer "github.com/buffo/arch-view/internal/analyzers/rust"
	tsanalyzer "github.com/buffo/arch-view/internal/analyzers/typescript"
	model "github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/quality"
	"github.com/buffo/arch-view/internal/viewer/layout"
)

const inventorySchemaVersion = "arch-view.docs-inventory/v1"

type inventory struct {
	SchemaVersion   string                       `json:"schema_version"`
	CLICommands     []cliCommand                 `json:"cli_commands"`
	Analyzers       []analysis.Manifest          `json:"analyzers"`
	MetricProviders []metricProvider             `json:"metric_providers"`
	QualityRules    []quality.RuleDescriptor     `json:"quality_rules"`
	Layout          layout.LayoutOptionsResponse `json:"layout"`
	Formats         formats                      `json:"formats"`
}

type cliCommand struct {
	Name       string   `json:"name"`
	HelpArgs   []string `json:"help_args"`
	DocsPath   string   `json:"docs_path"`
	DocsAnchor string   `json:"docs_anchor"`
}

type metricProvider struct {
	ID           string                         `json:"id"`
	Version      string                         `json:"version"`
	Capabilities []quality.CapabilityDescriptor `json:"capabilities"`
}

type formats struct {
	ModelSchema         string   `json:"model_schema"`
	QualitySchema       string   `json:"quality_schema"`
	BaselineSchema      string   `json:"baseline_schema"`
	LayoutSchema        string   `json:"layout_schema"`
	ModelFields         []string `json:"model_fields"`
	ProfileFields       []string `json:"quality_profile_fields"`
	RuleBindingFields   []string `json:"rule_binding_fields"`
	TypedBlockFields    []string `json:"typed_config_block_fields"`
	BaselineFields      []string `json:"baseline_fields"`
	BaselineEntryFields []string `json:"baseline_entry_fields"`
	ExportFormats       []string `json:"export_formats"`
	QualityStatuses     []string `json:"quality_statuses"`
	QualityCoverage     []string `json:"quality_coverage"`
	QualitySeverities   []string `json:"quality_severities"`
}

func main() {
	output := flag.String("output", "-", "JSON output file, or - for stdout")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "docs-inventory does not accept positional arguments")
		os.Exit(2)
	}

	value, err := buildInventory()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	data = append(data, '\n')
	if *output == "-" {
		_, _ = os.Stdout.Write(data)
		return
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*output, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func buildInventory() (inventory, error) {
	registry := analysis.NewRegistry()
	for _, analyzer := range []analysis.Analyzer{
		goanalyzer.New(),
		clojureanalyzer.New(),
		pyanalyzer.New(),
		rustanalyzer.New(),
		tsanalyzer.New(),
	} {
		if err := registry.Register(analyzer); err != nil {
			return inventory{}, err
		}
	}
	catalog := quality.NewDefaultCatalog()
	providers := make([]metricProvider, 0, len(catalog.ListMetricProviders()))
	for _, provider := range catalog.ListMetricProviders() {
		providers = append(providers, metricProvider{
			ID: provider.ID(), Version: provider.Version(), Capabilities: provider.Capabilities(),
		})
	}
	return inventory{
		SchemaVersion: inventorySchemaVersion,
		CLICommands: []cliCommand{
			{Name: "analyzers", HelpArgs: []string{"analyzers", "--help"}, DocsPath: "/cli/analyzers", DocsAnchor: "#options"},
			{Name: "analyze", HelpArgs: []string{"analyze", "--help"}, DocsPath: "/cli/analyze", DocsAnchor: "#required-options"},
			{Name: "open", HelpArgs: []string{"open", "--help"}, DocsPath: "/cli/open", DocsAnchor: "#input-options"},
			{Name: "export", HelpArgs: []string{"export", "--help"}, DocsPath: "/cli/export", DocsAnchor: "#required-options"},
			{Name: "quality baseline", HelpArgs: []string{"quality", "baseline", "--help"}, DocsPath: "/cli/quality-baseline", DocsAnchor: "#options"},
			{Name: "model normalize", HelpArgs: []string{"model", "normalize", "--help"}, DocsPath: "/cli/model", DocsAnchor: "#normalize"},
			{Name: "model validate", HelpArgs: []string{"model", "validate", "--help"}, DocsPath: "/cli/model", DocsAnchor: "#validate"},
			{Name: "model projection", HelpArgs: []string{"model", "projection", "--help"}, DocsPath: "/cli/model", DocsAnchor: "#projection"},
		},
		Analyzers:       registry.ListManifests(),
		MetricProviders: providers,
		QualityRules:    catalog.ListQualityRules(),
		Layout:          layout.Catalog(),
		Formats: formats{
			ModelSchema:         "arch-view.model/v1",
			QualitySchema:       quality.SchemaVersion,
			BaselineSchema:      quality.BaselineSchemaVersion,
			LayoutSchema:        layout.ConfigSchemaVersion,
			ModelFields:         jsonFieldNames(model.Model{}),
			ProfileFields:       jsonFieldNames(quality.QualityProfile{}),
			RuleBindingFields:   jsonFieldNames(quality.RuleBinding{}),
			TypedBlockFields:    jsonFieldNames(quality.TypedConfigBlock{}),
			BaselineFields:      jsonFieldNames(quality.Baseline{}),
			BaselineEntryFields: jsonFieldNames(quality.BaselineEntry{}),
			ExportFormats:       []string{"json", "html", "svg"},
			QualityStatuses:     []string{"active", "suppressed", "baseline", "resolved", "not_evaluable"},
			QualityCoverage:     []string{"observed", "absent", "unknown", "unsupported", "partial", "not_evaluable"},
			QualitySeverities:   []string{"info", "warning", "error", "blocker"},
		},
	}, nil
}

func jsonFieldNames(value any) []string {
	typeOfValue := reflect.TypeOf(value)
	if typeOfValue.Kind() == reflect.Pointer {
		typeOfValue = typeOfValue.Elem()
	}
	fields := make([]string, 0, typeOfValue.NumField())
	for index := 0; index < typeOfValue.NumField(); index++ {
		field := typeOfValue.Field(index)
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" {
			continue
		}
		fields = append(fields, name)
	}
	sort.Strings(fields)
	return fields
}

package profile

import (
	"context"
	"fmt"
	"sort"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
)

type registeredDiagnosticProvider struct {
	provider ports.DiagnosticProvider
	metadata ports.Extension
}

func (registry *Registry) RegisterDiagnosticProvider(provider ports.DiagnosticProvider) (err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("diagnostic provider registration failed: %v", failure)
		}
	}()
	metadata := provider.Metadata()
	metadata.Kind = "diagnostic_provider"
	metadata.ShapeDefinition = nil
	if !domain.ValidExtensionIdentity(metadata.ID, metadata.Version) {
		return fmt.Errorf("diagnostic provider requires namespaced ID and version")
	}
	if metadata.Description == "" || len(metadata.Capabilities) == 0 || metadata.DefinitionSchema == nil {
		return fmt.Errorf("diagnostic provider requires description, capabilities and output schema")
	}
	metadata, err = captureExtensionValue(metadata)
	if err != nil {
		return fmt.Errorf("diagnostic metadata: %w", err)
	}
	key := metadata.ID + "@" + metadata.Version
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if registry.diagnosticProviders == nil {
		registry.diagnosticProviders = make(map[string]registeredDiagnosticProvider)
	}
	if _, exists := registry.diagnosticProviders[key]; exists {
		return fmt.Errorf("diagnostic provider already registered: %s", key)
	}
	registry.diagnosticProviders[key] = registeredDiagnosticProvider{provider, metadata}
	return nil
}

func (registry *Registry) diagnosticEntries() []registeredDiagnosticProvider {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	entries := make([]registeredDiagnosticProvider, 0, len(registry.diagnosticProviders))
	for _, entry := range registry.diagnosticProviders {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i].metadata, entries[j].metadata
		if a.ID != b.ID {
			return a.ID < b.ID
		}
		return a.Version < b.Version
	})
	return entries
}

func (registry *Registry) DiagnosticProviderCatalog() []ports.Extension {
	result := make([]ports.Extension, 0)
	for _, entry := range registry.diagnosticEntries() {
		result = append(result, cloneExtensionMetadata(entry.metadata))
	}
	return result
}

func (registry *Registry) Diagnose(ctx context.Context, index domain.BundleIndex) ([]domain.Diagnostic, error) {
	var diagnostics []domain.Diagnostic
	for _, entry := range registry.diagnosticEntries() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		values, err := invokeDiagnosticProvider(ctx, entry.provider, index)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if err != nil {
			diagnostics = append(diagnostics, domain.Diagnostic{Code: "okf_diagnostic_provider_failed", Severity: "warning", Category: "extension", BundleID: index.BundleID, Message: "A registered diagnostic provider failed.", Details: map[string]any{"provider_id": entry.metadata.ID, "version": entry.metadata.Version, "reason": err.Error()}, Recovery: "Repair or unregister the diagnostic provider; built-in diagnostics remain available."})
			continue
		}
		for _, value := range values {
			value.BundleID = index.BundleID
			diagnostics = append(diagnostics, value)
		}
	}
	return domain.CloneDiagnostics(diagnostics), nil
}

func invokeDiagnosticProvider(ctx context.Context, provider ports.DiagnosticProvider, index domain.BundleIndex) (values []domain.Diagnostic, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			values = nil
			err = fmt.Errorf("provider failed: %v", failure)
		}
	}()
	values, err = provider.Diagnose(ctx, domain.CloneIndex(index))
	if err != nil {
		return nil, err
	}
	for _, value := range values {
		if value.Code == "" || value.Message == "" || value.Category == "" || (value.Severity != "info" && value.Severity != "warning" && value.Severity != "error") {
			return nil, fmt.Errorf("provider returned an invalid diagnostic")
		}
	}
	values, err = captureExtensionValue(values)
	if err != nil {
		return nil, fmt.Errorf("provider returned unencodable diagnostics: %w", err)
	}
	return domain.CloneDiagnostics(values), nil
}

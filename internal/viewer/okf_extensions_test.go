package viewer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/buffo/arch-view/internal/okf/application"
	"github.com/buffo/arch-view/internal/okf/contract"
	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/ports"
	"github.com/buffo/arch-view/internal/okf/profile"
)

type brokenHTTPRuleMetadata struct{ ports.RuleStrategy }

func (brokenHTTPRuleMetadata) ID() string          { return "test.metadata_failure" }
func (brokenHTTPRuleMetadata) Version() string     { return "1" }
func (brokenHTTPRuleMetadata) Description() string { panic("metadata failure") }

type brokenHTTPAdapterMetadata struct{}

func (brokenHTTPAdapterMetadata) ID() string      { panic("metadata failure") }
func (brokenHTTPAdapterMetadata) Version() string { return "1" }
func (brokenHTTPAdapterMetadata) Relationships(context.Context, domain.BundleIndex) ([]domain.Relationship, []domain.Diagnostic) {
	return nil, nil
}

func TestOKFExtensionMetadataFailuresUseHTTPFailureEnvelope(t *testing.T) {
	for _, kind := range []string{"rule", "relationship"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			registry := profile.NewRegistry()
			var adapters []ports.RelationshipAdapter
			wantCode := "okf_relationship_adapter_failed"
			if kind == "rule" {
				if err := registry.Register(brokenHTTPRuleMetadata{}); err != nil {
					t.Fatal(err)
				}
				wantCode = "okf_extension_catalog_failed"
			} else {
				adapters = append(adapters, brokenHTTPAdapterMetadata{})
			}
			api := application.NewWithDependencies(root, nil, nil, nil, registry, adapters...)
			server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: root, OKFApplication: api})
			if err != nil {
				t.Fatal(err)
			}
			host := httptest.NewServer(server.Handler())
			defer host.Close()
			response := getOKFHTTP(t, host.URL+"/v1/okf/extensions")
			var envelope struct {
				contract.Failure
				Data any `json:"data"`
			}
			decodeOKFHTTP(t, response, &envelope)
			if response.status != http.StatusInternalServerError || envelope.Error.Code != wantCode || envelope.Meta.RequestID == "" || envelope.Data != nil {
				t.Fatalf("failure response=%d %+v", response.status, envelope)
			}
			if next := getOKFHTTP(t, host.URL+"/v1/okf/profiles"); next.status != http.StatusOK {
				t.Fatalf("metadata failure poisoned subsequent request: %d %s", next.status, next.body)
			}
		})
	}
}

func TestOKFExtensionsHTTPPublishesBuiltInParameterSchemas(t *testing.T) {
	server, err := NewServer(fixtureModel(t), ServerOptions{SourceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	host := httptest.NewServer(server.Handler())
	defer host.Close()
	response := getOKFHTTP(t, host.URL+"/v1/okf/extensions")
	var envelope struct {
		Meta contract.Meta `json:"meta"`
		Data struct {
			Extensions []ports.Extension `json:"extensions"`
		} `json:"data"`
	}
	decodeOKFHTTP(t, response, &envelope)
	if response.status != http.StatusOK || len(envelope.Data.Extensions) != 12 {
		t.Fatalf("response=%d %+v", response.status, envelope)
	}
	values := make([]any, len(envelope.Data.Extensions))
	for index, value := range envelope.Data.Extensions {
		values[index] = value
	}
	wantRevision, err := contract.ExtensionCatalogRevision(values)
	if err != nil || envelope.Meta.Revision != wantRevision {
		t.Fatalf("revision does not identify returned catalog: %+v %v", envelope.Meta, err)
	}
	for _, extension := range envelope.Data.Extensions {
		if extension.Kind == "detail_renderer" {
			if extension.ID != "okf.detail.commonmark" || extension.Version != "1" || extension.ParameterSchema["type"] != "object" || extension.ParameterSchema["additionalProperties"] != false {
				t.Fatalf("invalid default detail descriptor: %+v", extension)
			}
			continue
		}
		if extension.Kind == "shape" {
			if extension.DefinitionSchema["$id"] != "urn:arch-view:okf:shape-definition:1" || extension.DefinitionSchema["type"] != "object" {
				t.Fatalf("missing versioned shape schema: %+v", extension)
			}
			if extension.ShapeDefinition == nil || extension.ShapeDefinition.ID != extension.ID || extension.ShapeDefinition.Version != extension.Version {
				t.Fatalf("missing registered shape definition: %+v", extension)
			}
			continue
		}
		if extension.Version != "1" || extension.Kind != "rule" || extension.ParameterSchema["type"] != "object" {
			t.Fatalf("missing rule schema metadata: %+v", extension)
		}
		properties, ok := extension.ParameterSchema["properties"].(map[string]any)
		if !ok || len(properties) == 0 {
			t.Fatalf("missing schema properties: %+v", extension)
		}
		if required, ok := extension.ParameterSchema["required"].([]any); !ok || len(required) == 0 {
			t.Fatalf("missing required parameters: %+v", extension)
		}
	}
}

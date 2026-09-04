package source

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/buffo/arch-view/internal/okf/domain"
	"github.com/buffo/arch-view/internal/okf/markdown"
	"gopkg.in/yaml.v3"
)

var knownFrontmatterKeys = map[string]bool{
	"type": true, "title": true, "description": true, "resource": true,
	"tags": true, "timestamp": true, "state": true, "role": true,
	"parent": true, "parents": true, "parent_id": true, "children": true,
	"child": true, "hierarchy": true, "relationships": true,
}

func parseConcept(pathValue string, data []byte) (domain.ConceptDocument, error) {
	if !utf8.Valid(data) {
		return domain.ConceptDocument{}, fmt.Errorf("concept source must be valid UTF-8")
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return domain.ConceptDocument{}, fmt.Errorf("missing YAML frontmatter")
	}
	closing := -1
	for index := 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) == "---" {
			closing = index
			break
		}
	}
	if closing < 0 {
		return domain.ConceptDocument{}, fmt.Errorf("frontmatter is not closed")
	}
	var frontmatterNode yaml.Node
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:closing], "\n")), &frontmatterNode); err != nil {
		return domain.ConceptDocument{}, fmt.Errorf("frontmatter YAML is invalid: %w", err)
	}
	frontmatter, err := yamlNodeMap(&frontmatterNode)
	if err != nil {
		return domain.ConceptDocument{}, err
	}
	typeValue, ok := frontmatter["type"].(string)
	if !ok || strings.TrimSpace(typeValue) == "" {
		return domain.ConceptDocument{}, fmt.Errorf("frontmatter requires a non-empty string type")
	}
	conceptID := trimMarkdownExtension(pathValue)
	document := domain.ConceptDocument{
		ConceptID:   conceptID,
		SourcePath:  pathValue,
		Type:        strings.TrimSpace(typeValue),
		Frontmatter: domain.CloneMap(frontmatter),
		Markdown:    strings.Join(lines[closing+1:], "\n"),
		Links:       extractLinks(strings.Join(lines[closing+1:], "\n")),
		Provenance: []domain.Provenance{{
			Source: "okf_document",
			Path:   pathValue,
		}},
	}
	document.Title, _ = frontmatter["title"].(string)
	document.Description, _ = frontmatter["description"].(string)
	document.Tags = stringSlice(frontmatter["tags"])
	document.ExplicitParents = append(document.ExplicitParents, stringSlice(frontmatter["parents"])...)
	document.ExplicitParents = append(document.ExplicitParents, stringSlice(frontmatter["parent"])...)
	document.ExplicitParents = append(document.ExplicitParents, stringSlice(frontmatter["parent_id"])...)
	document.ExplicitChildren = append(document.ExplicitChildren, stringSlice(frontmatter["children"])...)
	document.ExplicitChildren = append(document.ExplicitChildren, stringSlice(frontmatter["child"])...)
	document.UnknownFrontmatter = make(map[string]any)
	for key, value := range frontmatter {
		if !knownFrontmatterKeys[key] {
			document.UnknownFrontmatter[key] = domain.CloneMap(map[string]any{"value": value})["value"]
		}
	}
	if len(document.UnknownFrontmatter) == 0 {
		document.UnknownFrontmatter = nil
	}
	return document, nil
}

func yamlNodeMap(root *yaml.Node) (map[string]any, error) {
	if root.Kind == 0 || len(root.Content) == 0 {
		return map[string]any{}, nil
	}
	node := root.Content[0]
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("frontmatter must be a YAML mapping")
	}
	if err := validateYAMLKeys(node); err != nil {
		return nil, err
	}
	var result map[string]any
	// The YAML decoder handles aliases, merge keys, duplicate keys, and alias
	// expansion limits. Do not silently turn unsupported nodes into nil.
	if err := node.Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid frontmatter: %w", err)
	}
	if _, err := json.Marshal(result); err != nil {
		return nil, fmt.Errorf("frontmatter must contain JSON-compatible values: %w", err)
	}
	return result, nil
}

func validateYAMLKeys(node *yaml.Node) error {
	if node.Kind == yaml.MappingNode {
		for index := 0; index+1 < len(node.Content); index += 2 {
			key := node.Content[index]
			if key.Kind != yaml.ScalarNode || (key.Tag != "!!str" && key.Tag != "!!merge") || strings.TrimSpace(key.Value) == "" {
				return fmt.Errorf("frontmatter keys must be non-empty strings")
			}
		}
	}
	// Alias targets occur elsewhere in the tree. Decode checks recursive aliases.
	for _, child := range node.Content {
		if err := validateYAMLKeys(child); err != nil {
			return err
		}
	}
	return nil
}

func stringSlice(value any) []string {
	switch item := value.(type) {
	case string:
		if strings.TrimSpace(item) == "" {
			return nil
		}
		return []string{strings.TrimSpace(item)}
	case []any:
		result := make([]string, 0, len(item))
		for _, entry := range item {
			if text, ok := entry.(string); ok && strings.TrimSpace(text) != "" {
				result = append(result, strings.TrimSpace(text))
			}
		}
		return result
	case []string:
		return append([]string(nil), item...)
	default:
		return nil
	}
}

func extractLinks(value string) []domain.Link { return markdown.Extract(value) }

func trimMarkdownExtension(value string) string {
	if len(value) >= len(".md") && strings.EqualFold(value[len(value)-len(".md"):], ".md") {
		return value[:len(value)-len(".md")]
	}
	return value
}

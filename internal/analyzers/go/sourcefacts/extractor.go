// Package sourcefacts contains the registered Go implementation of the
// language-neutral source-facts extractor contract.
package sourcefacts

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/sourceindex"
	"github.com/buffo/arch-view/internal/analysis/syntax"
)

const (
	ExtractorID      = "extractor:go-source-facts"
	ExtractorVersion = "1.0.0"
)

// Extractor emits named top-level Go declarations, documentation candidates,
// visibility facts, and explicit file-to-symbol relations.
type Extractor struct{}

var _ sourceindex.SourceFactExtractor = Extractor{}

// NewRegistry returns the Go analyzer's local source-fact registry. The
// registry is intentionally owned by the analyzer instance, so adding another
// language or extractor does not mutate global process state.
func NewRegistry() *sourceindex.Registry {
	registry := sourceindex.NewRegistry()
	_ = registry.Register(Extractor{})
	return registry
}

func (Extractor) ID() string      { return ExtractorID }
func (Extractor) Version() string { return ExtractorVersion }

func (Extractor) Capabilities() []analysis.CapabilityDescriptor {
	return []analysis.CapabilityDescriptor{
		{ID: sourceindex.CapabilityDeclarations, Version: "v1", SupportedLanguages: []string{"go"}, Description: "Named top-level Go declarations and locations."},
		{ID: sourceindex.CapabilityDocumentation, Version: "v1", SupportedLanguages: []string{"go"}, Description: "Go documentation candidates and primary selection."},
		{ID: sourceindex.CapabilityVisibility, Version: "v1", SupportedLanguages: []string{"go"}, Description: "Go exported and unexported visibility facts."},
		{ID: sourceindex.CapabilityCallableMetrics, Version: "v1", SupportedLanguages: []string{"go"}, Description: "Go callable body spans, cyclomatic complexity, and nesting metrics."},
		{ID: sourceindex.CapabilitySolidStructure, Version: "1.0.0", SupportedLanguages: []string{"go"}, Description: "Go structural counts for advisory SOLID signals."},
	}
}

func (Extractor) Extract(input sourceindex.SourceFactInput) (sourceindex.FactBatch, error) {
	if input.Tree == nil {
		return sourceindex.FactBatch{}, fmt.Errorf("go source-fact extraction requires a syntax tree")
	}
	root := input.Tree
	targets := declarationTargets(root)
	comments := commentNodes(root)
	structuralByKey := map[string]structuralFacts{}
	if requestedCapability(input.RequestedCapabilities, sourceindex.CapabilitySolidStructure) {
		structuralByKey = collectStructuralFacts(input.File.Path, input.Content)
	}
	result := sourceindex.FactBatch{
		Symbols:       make([]analysis.SymbolRecord, 0, len(targets)),
		Documentation: make([]analysis.DocumentationRecord, 0, len(targets)),
		Relations:     make([]analysis.CodeRelation, 0, len(targets)*2),
	}
	for _, target := range targets {
		for _, name := range target.Names {
			if name.Value == "" || name.Value == "_" {
				continue
			}
			stableKey := fmt.Sprintf("%s:%s:%d", target.Kind, name.Value, name.Range.StartByte)
			span := sourceSpan(input.File, target.Node.Range())
			var bodySpan *analysis.SourceSpan
			if target.Category == analysis.SymbolCategoryCallable {
				if body := target.Node.ChildByFieldName("body"); body != nil {
					span := sourceSpan(input.File, body.Range())
					bodySpan = &span
				}
			}
			symbol := analysis.SymbolRecord{
				ID:            stableKey,
				Name:          name.Value,
				QualifiedName: qualifiedName(target, name.Value),
				Category:      target.Category,
				LanguageKind:  target.LanguageKind,
				Visibility:    visibility(name.Value, input.File, Extractor{}),
				Locations: []analysis.SymbolLocation{{
					Kind:               "location:declaration",
					Span:               span,
					SourceReferenceIDs: append([]string(nil), input.File.Provenance.EvidenceIDs...),
				}},
				BodySpan:      bodySpan,
				StableKey:     stableKey,
				IdentityBasis: "identity:declaration-span-name",
				Provenance:    provenance(analysis.FactStatusObserved, Extractor{}),
				Extensions:    []analysis.ExtensionBlock{},
			}
			if facts, ok := structuralByKey[stableKey]; ok {
				applyStructuralFacts(&symbol, facts)
			}
			result.Symbols = append(result.Symbols, symbol)
			result.Documentation = append(result.Documentation, documentationForSymbol(input.File, symbol, target.DocumentationAnchor, comments, input.Content, Extractor{}))
			fileRef := analysis.EntityRef{Kind: "file", ID: input.File.ID}
			symbolRef := analysis.EntityRef{Kind: "symbol", ID: stableKey}
			result.Relations = append(result.Relations,
				analysis.CodeRelation{
					ID:            "contains:" + stableKey,
					Category:      analysis.RelationContains,
					LanguageKind:  "relation:file-symbol",
					FromRef:       fileRef,
					ToRef:         entityRefPointer(symbolRef),
					EvidenceSpans: []analysis.SourceSpan{span},
					Provenance:    provenance(analysis.FactStatusObserved, Extractor{}),
					Extensions:    []analysis.ExtensionBlock{},
				},
				analysis.CodeRelation{
					ID:            "declares:" + stableKey,
					Category:      analysis.RelationDeclares,
					LanguageKind:  "relation:file-symbol",
					FromRef:       fileRef,
					ToRef:         entityRefPointer(symbolRef),
					EvidenceSpans: []analysis.SourceSpan{span},
					Provenance:    provenance(analysis.FactStatusObserved, Extractor{}),
					Extensions:    []analysis.ExtensionBlock{},
				},
			)
			if target.Category == analysis.SymbolCategoryCallable && requestedCapability(input.RequestedCapabilities, sourceindex.CapabilityCallableMetrics) && bodySpan != nil {
				complexity, nesting := goCallableMetrics(target.Node.ChildByFieldName("body"))
				metricSubject := analysis.EntityRef{Kind: "symbol", ID: stableKey}
				metricProvenance := provenance(analysis.FactStatusObserved, Extractor{})
				metricProvenance.EvidenceIDs = append([]string(nil), input.File.Provenance.EvidenceIDs...)
				decisionVocabulary := map[string]any{
					"nodes":             []string{"if_statement", "for_statement", "expression_case", "type_case", "communication_case"},
					"boolean_operators": []string{"&&", "||"},
					"default_cases":     false,
				}
				result.Metrics = append(result.Metrics,
					analysis.MetricFact{
						ID: metricID("complexity", stableKey), SubjectRef: metricSubject,
						MetricID: "source:callable.cyclomatic_complexity", Value: analysis.MetricValue{Kind: "integer", Value: complexity}, Unit: "unit:complexity",
						FormulaID: "formula:go.cyclomatic-complexity", FormulaVersion: ExtractorVersion,
						Provenance: metricProvenance,
						Extensions: []analysis.ExtensionBlock{
							{Namespace: "metric:go", SchemaVersion: ExtractorVersion, Capability: "decision-vocabulary", Payload: decisionVocabulary},
						},
					},
					analysis.MetricFact{
						ID: metricID("nesting", stableKey), SubjectRef: metricSubject,
						MetricID: "source:callable.max_nesting_depth", Value: analysis.MetricValue{Kind: "integer", Value: nesting}, Unit: "unit:depth",
						FormulaID: "formula:go.max-nesting-depth", FormulaVersion: ExtractorVersion,
						Provenance: metricProvenance,
						Extensions: []analysis.ExtensionBlock{
							{Namespace: "metric:go", SchemaVersion: ExtractorVersion, Capability: "control-flow-vocabulary", Payload: map[string]any{"nodes": []string{"if_statement", "for_statement", "expression_switch_statement", "type_switch_statement", "select_statement"}}},
						},
					},
				)
			}
		}
	}
	return result, nil
}

func requestedCapability(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func metricID(kind, stableKey string) string { return "go:" + kind + ":" + stableKey }

func goCallableMetrics(node syntax.Node) (int, int) {
	if node == nil {
		return 0, 0
	}
	decisions := 0
	syntax.Walk(node, func(value syntax.Node) bool {
		switch value.Type() {
		case "if_statement", "for_statement":
			decisions++
		case "expression_case", "type_case", "communication_case":
			if !isDefaultCase(value) {
				decisions++
			}
		case "binary_expression":
			if operator := binaryOperator(value); operator == "&&" || operator == "||" {
				decisions++
			}
		}
		return true
	})
	return 1 + decisions, maxGoNesting(node, 0)
}

func maxGoNesting(node syntax.Node, parent int) int {
	if node == nil {
		return parent
	}
	depth := parent
	switch node.Type() {
	case "if_statement", "for_statement", "expression_switch_statement", "type_switch_statement", "select_statement":
		depth++
	}
	maximum := depth
	for index := 0; index < node.ChildCount(); index++ {
		candidate := maxGoNesting(node.Child(index), depth)
		if candidate > maximum {
			maximum = candidate
		}
	}
	return maximum
}

func isDefaultCase(node syntax.Node) bool {
	if node == nil {
		return false
	}
	for index := 0; index < node.ChildCount(); index++ {
		child := node.Child(index)
		if child != nil && child.Type() == "default" {
			return true
		}
	}
	return strings.HasPrefix(strings.TrimSpace(node.Text()), "default")
}

func binaryOperator(node syntax.Node) string {
	if node == nil {
		return ""
	}
	for index := 0; index < node.ChildCount(); index++ {
		child := node.Child(index)
		if child != nil && !child.IsNamed() {
			return strings.TrimSpace(child.Text())
		}
	}
	return ""
}

type declarationTarget struct {
	Node                syntax.Node
	Kind                string
	Category            string
	LanguageKind        string
	Receiver            string
	Names               []declarationName
	DocumentationAnchor int
}

type declarationName struct {
	Value string
	Range syntax.Range
}

func declarationTargets(root syntax.Node) []declarationTarget {
	result := make([]declarationTarget, 0)
	for index := 0; index < root.ChildCount(); index++ {
		node := root.Child(index)
		if node == nil {
			continue
		}
		switch node.Type() {
		case "function_declaration", "method_declaration":
			nameNode := node.ChildByFieldName("name")
			if nameNode == nil {
				continue
			}
			kind := "go:function"
			category := analysis.SymbolCategoryCallable
			receiver := ""
			if node.Type() == "method_declaration" {
				kind = "go:method"
				receiver = receiverName(node.ChildByFieldName("receiver"))
			}
			result = append(result, declarationTarget{Node: node, Kind: kind, Category: category, LanguageKind: kind, Receiver: receiver, Names: []declarationName{{Value: nameNode.Text(), Range: nameNode.Range()}}, DocumentationAnchor: int(node.Range().StartByte)})
		case "type_declaration":
			collectDeclarationSpecs(node, int(node.Range().StartByte), &result)
		case "const_declaration", "var_declaration":
			collectValueSpecs(node, int(node.Range().StartByte), &result)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Node.Range().StartByte < result[j].Node.Range().StartByte })
	return result
}

func collectDeclarationSpecs(node syntax.Node, documentationAnchor int, result *[]declarationTarget) {
	if node == nil {
		return
	}
	if node.Type() == "type_spec" || node.Type() == "type_alias" {
		nameNode := node.ChildByFieldName("name")
		if nameNode == nil {
			return
		}
		kind := "go:type"
		if node.Type() == "type_alias" {
			kind = "go:type-alias"
		}
		if typeNode := node.ChildByFieldName("type"); typeNode != nil {
			kind = goTypeKind(kind, typeNode)
		}
		*result = append(*result, declarationTarget{
			Node:                node,
			Kind:                kind,
			Category:            analysis.SymbolCategoryType,
			LanguageKind:        kind,
			Names:               []declarationName{{Value: nameNode.Text(), Range: nameNode.Range()}},
			DocumentationAnchor: documentationAnchor,
		})
		return
	}
	for index := 0; index < node.NamedChildCount(); index++ {
		collectDeclarationSpecs(node.NamedChild(index), documentationAnchor, result)
	}
}

func collectValueSpecs(node syntax.Node, documentationAnchor int, result *[]declarationTarget) {
	if node == nil {
		return
	}
	if node.Type() == "const_spec" || node.Type() == "var_spec" {
		category := analysis.SymbolCategoryValue
		kind := "go:constant"
		if node.Type() == "var_spec" {
			kind = "go:variable"
		}
		*result = append(*result, declarationTarget{
			Node:                node,
			Kind:                kind,
			Category:            category,
			LanguageKind:        kind,
			Names:               valueNames(node),
			DocumentationAnchor: documentationAnchor,
		})
		return
	}
	for index := 0; index < node.NamedChildCount(); index++ {
		collectValueSpecs(node.NamedChild(index), documentationAnchor, result)
	}
}

func valueNames(node syntax.Node) []declarationName {
	result := make([]declarationName, 0)
	boundary := node.Range().EndByte
	if typeNode := node.ChildByFieldName("type"); typeNode != nil && typeNode.Range().StartByte < boundary {
		boundary = typeNode.Range().StartByte
	}
	if valueNode := node.ChildByFieldName("value"); valueNode != nil && valueNode.Range().StartByte < boundary {
		boundary = valueNode.Range().StartByte
	}
	for index := 0; index < node.NamedChildCount(); index++ {
		child := node.NamedChild(index)
		if child == nil || child.Type() != "identifier" || child.Range().EndByte > boundary {
			continue
		}
		result = append(result, declarationName{Value: child.Text(), Range: child.Range()})
	}
	if len(result) == 0 {
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			result = append(result, declarationName{Value: nameNode.Text(), Range: nameNode.Range()})
		}
	}
	return result
}

func goTypeKind(defaultKind string, node syntax.Node) string {
	switch node.Type() {
	case "struct_type":
		return "go:struct"
	case "interface_type":
		return "go:interface"
	default:
		return defaultKind
	}
}

func qualifiedName(target declarationTarget, name string) string {
	if target.Receiver != "" {
		return target.Receiver + "." + name
	}
	return name
}

func receiverName(node syntax.Node) string {
	if node == nil {
		return ""
	}
	value := strings.TrimSpace(node.Text())
	value = strings.TrimPrefix(value, "(")
	value = strings.TrimSuffix(value, ")")
	value = strings.TrimSpace(value)
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}
	value = fields[len(fields)-1]
	value = strings.TrimPrefix(value, "*")
	if index := strings.IndexByte(value, '['); index >= 0 {
		value = value[:index]
	}
	return strings.TrimSpace(value)
}

func visibility(name string, file analysis.FileRecord, extractor Extractor) analysis.VisibilityFact {
	classification := "unknown"
	languageValue := "unknown"
	if first := []rune(name); len(first) > 0 {
		if unicode.IsUpper(first[0]) {
			classification = "public"
			languageValue = "exported"
		} else {
			classification = "package"
			languageValue = "unexported"
		}
	}
	return analysis.VisibilityFact{
		Classification: classification,
		LanguageValue:  languageValue,
		Provenance:     provenance(analysis.FactStatusObserved, extractor),
	}
}

func documentationForSymbol(file analysis.FileRecord, symbol analysis.SymbolRecord, documentationAnchor int, comments []syntax.Node, content []byte, extractor Extractor) analysis.DocumentationRecord {
	commentGroup := nearestCommentGroup(symbol.Locations[0].Span.Start.ByteOffset, comments, content)
	if len(commentGroup) == 0 && documentationAnchor < symbol.Locations[0].Span.Start.ByteOffset {
		commentGroup = nearestCommentGroup(documentationAnchor, comments, content)
	}
	status := analysis.DocumentationAbsent
	rawText := ""
	normalizedText := ""
	spans := []analysis.SourceSpan{}
	if len(commentGroup) > 0 {
		status = analysis.DocumentationPresent
		parts := make([]string, 0, len(commentGroup))
		for _, comment := range commentGroup {
			parts = append(parts, cleanComment(comment.Text()))
			spans = append(spans, sourceSpan(file, comment.Range()))
		}
		rawText = strings.TrimSpace(strings.Join(parts, "\n"))
		normalizedText = strings.Join(strings.Fields(rawText), " ")
	}
	return analysis.DocumentationRecord{
		ID:                 "doc:" + symbol.StableKey,
		SubjectRef:         analysis.EntityRef{Kind: "symbol", ID: symbol.ID},
		SelectionGroup:     "go:doc",
		Format:             "go:doc",
		RawText:            rawText,
		NormalizedText:     normalizedText,
		Spans:              spans,
		SourceReferenceIDs: append([]string(nil), file.Provenance.EvidenceIDs...),
		AttachmentBasis:    "language-native",
		PrecedenceRank:     intPointer(0),
		IsPrimary:          true,
		Status:             status,
		Completeness:       analysis.DocumentationComplete,
		Provenance:         provenance(documentationFactStatus(status), extractor),
		Extensions:         []analysis.ExtensionBlock{},
	}
}

func commentNodes(root syntax.Node) []syntax.Node {
	result := make([]syntax.Node, 0)
	syntax.Walk(root, func(node syntax.Node) bool {
		if node != nil && node.Type() == "comment" {
			result = append(result, node)
		}
		return true
	})
	sort.Slice(result, func(i, j int) bool { return result[i].Range().StartByte < result[j].Range().StartByte })
	return result
}

func nearestCommentGroup(startByte int, comments []syntax.Node, content []byte) []syntax.Node {
	end := -1
	for index := range comments {
		if int(comments[index].Range().EndByte) <= startByte {
			end = index
			continue
		}
		break
	}
	if end < 0 {
		return nil
	}
	result := []syntax.Node{comments[end]}
	for index := end - 1; index >= 0; index-- {
		gapStart := int(comments[index].Range().EndByte)
		gapEnd := int(result[0].Range().StartByte)
		if !commentGap(content, gapStart, gapEnd) {
			break
		}
		result = append([]syntax.Node{comments[index]}, result...)
	}
	return result
}

func commentGap(content []byte, start, end int) bool {
	if start < 0 || end < start || end > len(content) {
		return false
	}
	newlines := 0
	for _, value := range content[start:end] {
		if value == '\n' {
			newlines++
		}
		if value != ' ' && value != '\t' && value != '\r' && value != '\n' {
			return false
		}
	}
	return newlines <= 1
}

func cleanComment(value string) string {
	value = strings.TrimSpace(value)
	switch {
	case strings.HasPrefix(value, "//"):
		return strings.TrimSpace(strings.TrimPrefix(value, "//"))
	case strings.HasPrefix(value, "/*") && strings.HasSuffix(value, "*/"):
		value = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "/*"), "*/"))
		lines := strings.Split(value, "\n")
		for index := range lines {
			lines[index] = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[index]), "*"))
		}
		return strings.TrimSpace(strings.Join(lines, "\n"))
	default:
		return value
	}
}

func sourceSpan(file analysis.FileRecord, value syntax.Range) analysis.SourceSpan {
	return analysis.SourceSpan{
		FileID: file.ID,
		Start: analysis.SpanPosition{
			ByteOffset: int(value.StartByte),
			Line:       int(value.Start.Row) + 1,
			Column:     int(value.Start.Column) + 1,
		},
		End: analysis.SpanPosition{
			ByteOffset: int(value.EndByte),
			Line:       int(value.End.Row) + 1,
			Column:     int(value.End.Column) + 1,
		},
		CoordinateSystem: analysis.SourceSpanCoordinateSystem,
		ContentHash:      file.Size.ContentHash,
	}
}

func provenance(status string, extractor Extractor) analysis.FactProvenance {
	return analysis.FactProvenance{Status: status, Basis: "syntax", EvidenceIDs: []string{}, Provider: extractor.ID(), ProviderVersion: extractor.Version()}
}

func documentationFactStatus(status string) string {
	switch status {
	case analysis.DocumentationPresent:
		return analysis.FactStatusObserved
	case analysis.DocumentationAbsent:
		return analysis.FactStatusAbsent
	case analysis.DocumentationUnknown:
		return analysis.FactStatusUnknown
	case analysis.DocumentationUnsupported:
		return analysis.FactStatusUnsupported
	case analysis.DocumentationPartial:
		return analysis.FactStatusPartial
	default:
		return analysis.FactStatusObserved
	}
}

func entityRefPointer(value analysis.EntityRef) *analysis.EntityRef { return &value }

func intPointer(value int) *int { return &value }

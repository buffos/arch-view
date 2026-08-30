package analysis

import (
	"sort"
	"strconv"
	"strings"
)

// CanonicalizeSourceIndexFacts applies the public source-index ordering contract:
// files by path, and structural facts by subject/path, primary byte offset,
// semantic kind/name, then opaque identity.
func CanonicalizeSourceIndexFacts(snapshot *SourceIndexSnapshot) {
	if snapshot == nil {
		return
	}
	paths := sourceIndexPaths(snapshot.Files)
	sort.Slice(snapshot.Files, func(i, j int) bool { return fileFactKey(snapshot.Files[i]) < fileFactKey(snapshot.Files[j]) })
	sort.Slice(snapshot.Symbols, func(i, j int) bool {
		return symbolFactKey(snapshot.Symbols[i], paths) < symbolFactKey(snapshot.Symbols[j], paths)
	})
	sort.Slice(snapshot.Documentation, func(i, j int) bool {
		return documentationFactKey(snapshot.Documentation[i], paths) < documentationFactKey(snapshot.Documentation[j], paths)
	})
	sort.Slice(snapshot.Occurrences, func(i, j int) bool {
		return occurrenceFactKey(snapshot.Occurrences[i], paths) < occurrenceFactKey(snapshot.Occurrences[j], paths)
	})
	sort.Slice(snapshot.Relations, func(i, j int) bool {
		return relationFactKey(snapshot.Relations[i], paths) < relationFactKey(snapshot.Relations[j], paths)
	})
	sort.Slice(snapshot.Metrics, func(i, j int) bool { return metricFactKey(snapshot.Metrics[i]) < metricFactKey(snapshot.Metrics[j]) })
}

func sourceIndexPaths(files []FileRecord) map[string]string {
	paths := make(map[string]string, len(files))
	for _, file := range files {
		paths[file.ID] = file.Path
	}
	return paths
}

func fileFactKey(file FileRecord) string {
	return strings.Join([]string{file.Path, file.ID}, "\x00")
}

func symbolFactKey(symbol SymbolRecord, paths map[string]string) string {
	path, offset := primarySpanKey(symbol.Locations, paths)
	return strings.Join([]string{path, byteOrderKey(offset), symbol.Category, symbol.LanguageKind, symbol.Name, symbol.ID}, "\x00")
}

func documentationFactKey(value DocumentationRecord, paths map[string]string) string {
	path, offset := firstSpanKey(value.Spans, paths)
	return strings.Join([]string{entityOrderKey(value.SubjectRef), path, byteOrderKey(offset), value.SelectionGroup, value.Format, value.ID}, "\x00")
}

func occurrenceFactKey(value SymbolOccurrence, paths map[string]string) string {
	return strings.Join([]string{paths[value.SourceSpan.FileID], byteOrderKey(value.SourceSpan.Start.ByteOffset), value.OccurrenceKind, value.TargetName, value.ID}, "\x00")
}

func relationFactKey(value CodeRelation, paths map[string]string) string {
	path, offset := firstSpanKey(value.EvidenceSpans, paths)
	target := ""
	if value.ToRef != nil {
		target = entityOrderKey(*value.ToRef)
	} else if value.UnresolvedTarget != nil {
		target = strings.Join([]string{value.UnresolvedTarget.QualifiedName, value.UnresolvedTarget.DisplayName, value.UnresolvedTarget.LanguageKind}, "\x00")
	}
	return strings.Join([]string{entityOrderKey(value.FromRef), target, path, byteOrderKey(offset), value.Category, value.LanguageKind, value.ID}, "\x00")
}

func metricFactKey(value MetricFact) string {
	return strings.Join([]string{entityOrderKey(value.SubjectRef), value.MetricID, value.FormulaID, value.FormulaVersion, value.ID}, "\x00")
}

func primarySpanKey(locations []SymbolLocation, paths map[string]string) (string, int) {
	if len(locations) == 0 {
		return "", 0
	}
	spans := make([]SourceSpan, len(locations))
	for index := range locations {
		spans[index] = locations[index].Span
	}
	return firstSpanKey(spans, paths)
}

func firstSpanKey(spans []SourceSpan, paths map[string]string) (string, int) {
	if len(spans) == 0 {
		return "", 0
	}
	bestPath := paths[spans[0].FileID]
	bestOffset := spans[0].Start.ByteOffset
	for _, span := range spans[1:] {
		candidatePath := paths[span.FileID]
		if candidatePath < bestPath || candidatePath == bestPath && span.Start.ByteOffset < bestOffset {
			bestPath = candidatePath
			bestOffset = span.Start.ByteOffset
		}
	}
	return bestPath, bestOffset
}

func entityOrderKey(value EntityRef) string {
	return strings.Join([]string{value.ScopeID, value.SnapshotID, value.Kind, value.ID}, "\x00")
}

func byteOrderKey(value int) string {
	return strings.Repeat("0", 20-len(strconv.Itoa(value))) + strconv.Itoa(value)
}

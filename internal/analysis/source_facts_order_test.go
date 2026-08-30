package analysis

import "testing"

func TestCanonicalizeSourceIndexFactsUsesPathsAndStructuralPositions(t *testing.T) {
	snapshot := SourceIndexSnapshot{
		Files: []FileRecord{{ID: "file-a", Path: "z.go"}, {ID: "file-z", Path: "a.go"}},
		Symbols: []SymbolRecord{
			{ID: "symbol-a", Name: "Later", Category: SymbolCategoryValue, Locations: []SymbolLocation{{Span: SourceSpan{FileID: "file-z", Start: SpanPosition{ByteOffset: 20}}}}},
			{ID: "symbol-z", Name: "Earlier", Category: SymbolCategoryValue, Locations: []SymbolLocation{{Span: SourceSpan{FileID: "file-z", Start: SpanPosition{ByteOffset: 2}}}}},
		},
	}
	CanonicalizeSourceIndexFacts(&snapshot)
	if snapshot.Files[0].Path != "a.go" || snapshot.Files[1].Path != "z.go" {
		t.Fatalf("canonical files = %#v", snapshot.Files)
	}
	if snapshot.Symbols[0].Name != "Earlier" || snapshot.Symbols[1].Name != "Later" {
		t.Fatalf("canonical symbols = %#v", snapshot.Symbols)
	}
}

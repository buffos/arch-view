package scanner

import (
	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/analysis/syntax"
)

type Project struct {
	Root                  string
	ModuleRoot            string
	ModulePath            string
	GoVersion             string
	WorkspacePath         string
	RelativeModuleRoot    string
	RelativeWorkspacePath string
	Boundary              string
}

type Package struct {
	Directory   string
	RelativeDir string
	ImportPath  string
	PackageName string
	Files       []File
}

type File struct {
	RelativePath    string
	PackageName     string
	SourceReference analysis.SourceReference
	Imports         []Import
	IsTest          bool
	IsGenerated     bool
	Constraints     []string
}

type Import struct {
	ImportPath  string
	Source      analysis.SourceReference
	Constraints []string
}

type ImportRecord struct {
	FromImportPath string
	Import         Import
}

// SourceFile retains every eligible readable source file independently of
// package parsing. This is the input boundary for the optional source-facts
// attachment; parser failures update AnalysisStatus rather than deleting the
// file from the source set.
type SourceFile struct {
	RelativePath       string
	Content            []byte
	Language           analysis.LanguageRef
	Roles              []string
	AnalysisStatus     string
	ModuleID           string
	SourceReferenceIDs []string
}

type ScanResult struct {
	Packages         []*Package
	SourceReferences []analysis.SourceReference
	Imports          []ImportRecord
	Diagnostics      []analysis.Diagnostic
	SourceFiles      []SourceFile
	SyntaxProvider   syntax.Provider
}

package scanner

import "github.com/buffo/arch-view/internal/analysis"

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

type ScanResult struct {
	Packages         []*Package
	SourceReferences []analysis.SourceReference
	Imports          []ImportRecord
	Diagnostics      []analysis.Diagnostic
}

type ImportRecord struct {
	FromImportPath string
	Import         Import
}

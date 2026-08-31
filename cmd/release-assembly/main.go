package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/buffo/arch-view/internal/analysis/distribution"
	appversion "github.com/buffo/arch-view/internal/version"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	repositoryRoot := flag.String("repository-root", "", "repository root containing go.mod and commands")
	releaseRoot := flag.String("release-root", "", "final release output root")
	platform := flag.String("platform", "", "exact target platform: windows-amd64, linux-amd64, or darwin-arm64")
	version := flag.String("version", "", "application release version; defaults to "+appversion.DefaultVersion)
	commit := flag.String("commit", "", "source commit identifier; defaults to unknown")
	buildDate := flag.String("build-date", "", "UTC build timestamp; defaults to unknown")
	buildID := flag.String("build-id", "", "controlled build identifier written to index.json")
	goCommand := flag.String("go-command", "go", "explicit Go toolchain command or path")
	ccCommand := flag.String("cc-command", "", "explicit C compiler command or path used by CGO")
	cxxCommand := flag.String("cxx-command", "", "explicit C++ compiler command or path used by CGO")
	analyzers := flag.String("analyzers", strings.Join(distribution.DefaultAnalyzerIDs(), ","), "comma-separated enabled analyzer IDs")
	flag.Parse()

	if strings.TrimSpace(*repositoryRoot) == "" || strings.TrimSpace(*releaseRoot) == "" || strings.TrimSpace(*platform) == "" || strings.TrimSpace(*buildID) == "" {
		return fmt.Errorf("repository-root, release-root, platform, and build-id are required")
	}
	return distribution.AssembleRelease(context.Background(), distribution.ReleaseOptions{
		RepositoryRoot: strings.TrimSpace(*repositoryRoot),
		OutputRoot:     *releaseRoot,
		Platform:       *platform,
		Version:        *version,
		Commit:         *commit,
		BuildDate:      *buildDate,
		BuildID:        *buildID,
		GoCommand:      *goCommand,
		CCCommand:      *ccCommand,
		CXXCommand:     *cxxCommand,
		AnalyzerIDs:    strings.Split(*analyzers, ","),
	})
}

// Package version owns the application release identity. Protocol, model,
// analyzer, and quality versions remain in their respective packages because
// they describe independently versioned contracts.
package version

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

const (
	Application    = "arch-view"
	DefaultVersion = "0.1.0"
	DefaultCommit  = "unknown"
	DefaultDate    = "unknown"
	DefaultBuildID = "local"

	linkerPackage = "github.com/buffo/arch-view/internal/version"
)

var (
	// These variables are replaced by release builds with Go linker flags. The
	// checked-in defaults make local source builds useful and identifiable.
	Version   = DefaultVersion
	Commit    = DefaultCommit
	BuildDate = DefaultDate
	BuildID   = DefaultBuildID

	semanticVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$`)
)

// Info is the application-level identity carried by a build.
type Info struct {
	Application string `json:"application"`
	Version     string `json:"version"`
	Commit      string `json:"commit"`
	BuildDate   string `json:"build_date"`
	BuildID     string `json:"build_id"`
}

// Current returns the application identity embedded in this process.
func Current() Info {
	return Info{
		Application: Application,
		Version:     Version,
		Commit:      Commit,
		BuildDate:   BuildDate,
		BuildID:     BuildID,
	}
}

// String formats application identity for human-readable CLI output.
func (info Info) String() string {
	return fmt.Sprintf("%s %s (commit %s, built %s, build %s)", info.Application, info.Version, info.Commit, info.BuildDate, info.BuildID)
}

// LinkerFlags returns the -X arguments used to embed release metadata in the
// host executable. The caller supplies the surrounding -buildid flag.
func (info Info) LinkerFlags() string {
	return strings.Join([]string{
		"-X=" + linkerPackage + ".Version=" + info.Version,
		"-X=" + linkerPackage + ".Commit=" + info.Commit,
		"-X=" + linkerPackage + ".BuildDate=" + info.BuildDate,
		"-X=" + linkerPackage + ".BuildID=" + info.BuildID,
	}, " ")
}

// Validate checks the values that are safe and meaningful in a release
// manifest and in linker arguments.
func (info Info) Validate() error {
	if info.Application != Application {
		return fmt.Errorf("application must be %q", Application)
	}
	if !semanticVersionPattern.MatchString(info.Version) {
		return fmt.Errorf("version must be semantic-version text: %q", info.Version)
	}
	fields := []struct {
		name  string
		value string
	}{
		{name: "commit", value: info.Commit},
		{name: "build date", value: info.BuildDate},
		{name: "build ID", value: info.BuildID},
	}
	for _, field := range fields {
		if strings.TrimSpace(field.value) == "" || strings.TrimSpace(field.value) != field.value {
			return fmt.Errorf("%s must be non-empty normalized text", field.name)
		}
		if strings.IndexFunc(field.value, unicode.IsSpace) >= 0 || strings.ContainsAny(field.value, "\x00\r\n") {
			return fmt.Errorf("%s contains unsupported whitespace or control characters", field.name)
		}
	}
	return nil
}

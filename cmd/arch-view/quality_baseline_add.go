package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/quality"
	qualitypolicy "github.com/buffo/arch-view/internal/quality/policy"
)

type managedBaselineCLIResult struct {
	Status               string   `json:"status"`
	BaselineFile         string   `json:"baseline_file"`
	ProfileFile          string   `json:"profile_file"`
	BaselineID           string   `json:"baseline_id"`
	Revision             string   `json:"revision"`
	AddedFindingKeys     []string `json:"added_finding_keys"`
	ExistingFindingKeys  []string `json:"existing_finding_keys"`
	ReevaluationRequired bool     `json:"reevaluation_required"`
}

func runQualityBaselineAdd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("arch-view quality baseline add", flag.ContinueOnError)
	fs.SetOutput(stderr)
	project := fs.String("project", "", "project root containing quality-profiles and quality-baselines")
	profilePath := fs.String("profile", "", "managed quality profile JSON file")
	baselineFile := fs.String("baseline-file", "", "direct JSON file name within quality-baselines")
	input := fs.String("input", "", "analysis or quality-report JSON file")
	baselineID := fs.String("baseline-id", "", "baseline identity when the profile has no baseline reference")
	revision := fs.String("revision", "", "explicit next baseline revision")
	expectedRevision := fs.String("expected-revision", "", "expected current baseline revision")
	reason := fs.String("reason", "", "reason recorded for each newly added entry")
	owner := fs.String("owner", "", "optional owner recorded for each entry")
	allActive := fs.Bool("all-active", false, "add every active finding with observed coverage")
	var findings stringList
	fs.Var(&findings, "finding", "active finding ID or stable key; repeatable")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			printQualityUsage(stdout)
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "quality baseline add does not accept positional arguments", map[string]any{"arguments": fs.Args()})
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if strings.TrimSpace(*project) == "" || strings.TrimSpace(*profilePath) == "" || strings.TrimSpace(*baselineFile) == "" || strings.TrimSpace(*input) == "" || strings.TrimSpace(*reason) == "" {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "quality baseline add requires --project, --profile, --baseline-file, --input, and --reason", nil)
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if *allActive && len(findings) > 0 {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "--all-active cannot be combined with --finding", nil)
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	selected := append([]string(nil), findings...)
	if !*allActive && len(selected) == 0 {
		err := analysis.NewHostError(analysis.ErrInvalidRequest, "quality baseline add requires at least one --finding or --all-active", nil)
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	root, err := filepath.Abs(*project)
	if err != nil {
		wrapped := analysis.WrapHostError(analysis.ErrInvalidOptions, "project root could not be normalized", err, nil)
		writeError(stderr, wrapped)
		return analysis.ExitCodeForError(wrapped)
	}
	store, err := qualitypolicy.NewFileStore(root)
	if err != nil {
		wrapped := analysis.WrapHostError(analysis.ErrInvalidOptions, "quality policy store could not be opened", err, nil)
		writeError(stderr, wrapped)
		return analysis.ExitCodeForError(wrapped)
	}
	profileFileName, err := managedQualityFileName(store.Root(), *profilePath, qualitypolicy.ProfileDirectoryName, false)
	if err != nil {
		wrapped := analysis.WrapHostError(analysis.ErrInvalidOptions, "quality profile path is not managed by the project", err, nil)
		writeError(stderr, wrapped)
		return analysis.ExitCodeForError(wrapped)
	}
	baselineFileName, err := managedQualityFileName(store.Root(), *baselineFile, qualitypolicy.BaselineDirectoryName, true)
	if err != nil {
		wrapped := analysis.WrapHostError(analysis.ErrInvalidOptions, "quality baseline path is not managed by the project", err, nil)
		writeError(stderr, wrapped)
		return analysis.ExitCodeForError(wrapped)
	}
	profile, err := store.ReadProfile(nil, profileFileName)
	if err != nil {
		wrapped := analysis.WrapHostError(analysis.ErrInvalidOptions, "quality profile could not be read", err, map[string]any{"profile": profileFileName})
		writeError(stderr, wrapped)
		return analysis.ExitCodeForError(wrapped)
	}
	service := qualitypolicy.NewService(store, quality.NewDefaultCatalog())
	profile, err = service.ValidateProfile(profile)
	if err != nil {
		wrapped := analysis.WrapHostError(analysis.ErrInvalidOptions, "quality profile is invalid", err, map[string]any{"profile": profileFileName})
		writeError(stderr, wrapped)
		return analysis.ExitCodeForError(wrapped)
	}
	report, err := readQualityReport(*input)
	if err != nil {
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if report.ProfileID != profile.ProfileID || report.ProfileVersion != profile.ProfileVersion {
		err := analysis.NewHostError(analysis.ErrInvalidOptions, "quality report does not match the selected profile", map[string]any{"report_profile": report.ProfileID + "@" + report.ProfileVersion, "profile": profile.ProfileID + "@" + profile.ProfileVersion})
		writeError(stderr, err)
		return analysis.ExitCodeForError(err)
	}
	if *allActive {
		for _, finding := range report.Findings {
			if finding.Status == quality.StatusActive {
				selected = append(selected, finding.ID)
			}
		}
	}
	entries := make([]quality.BaselineEntry, 0, len(selected))
	seen := make(map[string]struct{}, len(selected))
	for _, findingID := range selected {
		finding, ok := findQualityFinding(report, findingID)
		if !ok {
			err := analysis.NewHostError(analysis.ErrInvalidOptions, "quality baseline finding was not found in the report", map[string]any{"finding": findingID})
			writeError(stderr, err)
			return analysis.ExitCodeForError(err)
		}
		if finding.Status != quality.StatusActive {
			err := analysis.NewHostError(analysis.ErrInvalidOptions, "only active findings may be added to a baseline", map[string]any{"finding": findingID, "status": finding.Status})
			writeError(stderr, err)
			return analysis.ExitCodeForError(err)
		}
		if !qualityRuleCoverageObserved(report, finding.RuleID, finding.RuleVersion) {
			err := analysis.NewHostError(analysis.ErrInvalidOptions, "baseline entries require observed rule coverage", map[string]any{"finding": finding.FindingKey, "rule_id": finding.RuleID, "rule_version": finding.RuleVersion})
			writeError(stderr, err)
			return analysis.ExitCodeForError(err)
		}
		entry, entryErr := quality.CreateBaselineEntry(report, finding.ID, *reason, *owner)
		if entryErr != nil {
			wrapped := analysis.WrapHostError(analysis.ErrInvalidOptions, "quality baseline entry could not be created", entryErr, map[string]any{"finding": findingID})
			writeError(stderr, wrapped)
			return analysis.ExitCodeForError(wrapped)
		}
		identity := entry.FindingKey + "\x00" + entry.RuleID + "\x00" + entry.RuleVersion + "\x00" + entry.ProfileID + "\x00" + entry.ProfileVersion
		if _, duplicate := seen[identity]; duplicate {
			continue
		}
		seen[identity] = struct{}{}
		entries = append(entries, entry)
	}
	result, err := service.AppendBaseline(nil, qualitypolicy.BaselineAppendRequest{
		Profile: profile, ProfileFileName: profileFileName, BaselineFileName: baselineFileName,
		BaselineID: strings.TrimSpace(*baselineID), Entries: entries, Revision: strings.TrimSpace(*revision), ExpectedRevision: strings.TrimSpace(*expectedRevision),
	})
	if err != nil {
		wrapped := analysis.WrapHostError(analysis.ErrInvalidOptions, "quality baseline could not be updated", err, map[string]any{"error": err.Error()})
		writeError(stderr, wrapped)
		return analysis.ExitCodeForError(wrapped)
	}
	status := "unchanged"
	if result.Changed {
		status = "updated"
	}
	output := managedBaselineCLIResult{
		Status: status, BaselineFile: filepath.ToSlash(filepath.Join(qualitypolicy.BaselineDirectoryName, baselineFileName)), ProfileFile: filepath.ToSlash(filepath.Join(qualitypolicy.ProfileDirectoryName, profileFileName)),
		BaselineID: result.Baseline.BaselineID, Revision: result.Baseline.Revision, AddedFindingKeys: baselineEntryKeys(result.Added), ExistingFindingKeys: baselineEntryKeys(result.Existing), ReevaluationRequired: result.Changed,
	}
	return writeJSON(stdout, output)
}

func managedQualityFileName(root, value, directoryName string, allowMissing bool) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("file name is required")
	}
	baseDirectory := filepath.Join(root, directoryName)
	candidate := value
	if !filepath.IsAbs(candidate) {
		normalized := filepath.FromSlash(candidate)
		if filepath.Dir(normalized) == directoryName {
			candidate = filepath.Join(root, normalized)
		} else if filepath.Dir(normalized) == "." {
			candidate = filepath.Join(baseDirectory, normalized)
		} else {
			return "", fmt.Errorf("file must be a direct document in %s", directoryName)
		}
	}
	candidate, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	if !pathWithinDirectory(baseDirectory, candidate) || filepath.Dir(candidate) != filepath.Clean(baseDirectory) {
		return "", fmt.Errorf("file must be a direct document in %s", directoryName)
	}
	if !strings.EqualFold(filepath.Ext(candidate), ".json") {
		return "", fmt.Errorf("file must have a .json extension")
	}
	if info, statErr := os.Lstat(candidate); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("file may not be a symlink")
		}
		resolved, resolveErr := filepath.EvalSymlinks(candidate)
		if resolveErr != nil || !pathWithinDirectory(baseDirectory, resolved) {
			return "", fmt.Errorf("file resolves outside %s", directoryName)
		}
	} else if !allowMissing || !os.IsNotExist(statErr) {
		return "", statErr
	}
	return filepath.Base(candidate), nil
}

func pathWithinDirectory(root, candidate string) bool {
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(candidate))
	if err != nil {
		return false
	}
	return relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func findQualityFinding(report quality.QualityEvaluation, value string) (quality.QualityFinding, bool) {
	value = strings.TrimSpace(value)
	for _, finding := range report.Findings {
		if finding.ID == value || finding.FindingKey == value {
			return finding, true
		}
	}
	return quality.QualityFinding{}, false
}

func qualityRuleCoverageObserved(report quality.QualityEvaluation, ruleID, version string) bool {
	found := false
	for _, coverage := range report.Coverage {
		if coverage.RuleID != ruleID || coverage.RuleVersion != version {
			continue
		}
		found = true
		if coverage.Status != quality.CoverageObserved {
			return false
		}
	}
	return found
}

func baselineEntryKeys(entries []quality.BaselineEntry) []string {
	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		result = append(result, entry.FindingKey)
	}
	sort.Strings(result)
	return result
}

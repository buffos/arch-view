package main

import (
	"flag"
	"strings"
)

type analyzerCLIFlags struct {
	module           *string
	crate            *string
	features         *stringList
	target           *string
	includeTests     *bool
	includeExamples  *bool
	includeGenerated *bool
	includeExternal  *bool
	safeMode         *bool
	buildTags        *stringList
	excludes         *stringList
	sourceRoots      *stringList
	pythonVersion    *string
	includeStubs     *bool
	platform         *string
}

// collectAnalyzerCLIOptions sends only explicitly supplied analyzer options
// to the selected adapter. This keeps generic CLI defaults from becoming
// unsupported options for another language.
func collectAnalyzerCLIOptions(fs *flag.FlagSet, flags analyzerCLIFlags) map[string]any {
	options := map[string]any{}
	if flagWasSet(fs, "include-tests") {
		options["include_tests"] = *flags.includeTests
	}
	if flagWasSet(fs, "include-generated") {
		options["include_generated"] = *flags.includeGenerated
	}
	if flagWasSet(fs, "include-external") {
		options["include_external"] = *flags.includeExternal
	}
	if flagWasSet(fs, "safe-mode") {
		options["safe_mode"] = *flags.safeMode
	}
	if flagWasSet(fs, "module") && *flags.module != "" {
		options["module"] = *flags.module
	}
	if flagWasSet(fs, "crate") && *flags.crate != "" {
		options["crate"] = *flags.crate
	}
	if flags.features != nil && len(*flags.features) > 0 {
		options["features"] = []string(*flags.features)
	}
	if flagWasSet(fs, "target") && *flags.target != "" {
		options["target"] = *flags.target
	}
	if flagWasSet(fs, "include-examples") {
		options["include_examples"] = *flags.includeExamples
	}
	if len(*flags.buildTags) > 0 {
		options["build_tags"] = []string(*flags.buildTags)
	}
	if len(*flags.excludes) > 0 {
		options["exclude"] = []string(*flags.excludes)
	}
	if len(*flags.sourceRoots) > 0 {
		options["source_roots"] = []string(*flags.sourceRoots)
	}
	if flagWasSet(fs, "python-version") && *flags.pythonVersion != "" {
		options["python_version"] = *flags.pythonVersion
	}
	if flagWasSet(fs, "include-stubs") {
		options["include_stubs"] = *flags.includeStubs
	}
	if flagWasSet(fs, "platform") && strings.TrimSpace(*flags.platform) != "" {
		options["platform"] = strings.TrimSpace(*flags.platform)
	}
	return options
}

func flagWasSet(fs *flag.FlagSet, name string) bool {
	set := false
	fs.Visit(func(value *flag.Flag) {
		if value.Name == name {
			set = true
		}
	})
	return set
}

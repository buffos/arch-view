package main

import "flag"

type analyzerCLIFlags struct {
	module           *string
	includeTests     *bool
	includeGenerated *bool
	includeExternal  *bool
	safeMode         *bool
	buildTags        *stringList
	excludes         *stringList
	sourceRoots      *stringList
	pythonVersion    *string
	includeStubs     *bool
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

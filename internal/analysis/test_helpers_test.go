package analysis

import "context"

type fakeAnalyzer struct {
	manifest Manifest
	detect   func(context.Context, DetectRequest) (DetectionCandidate, error)
	analyze  func(context.Context, AnalyzeRequest) (AnalysisResult, error)
}

func (f *fakeAnalyzer) Manifest() Manifest {
	return f.manifest
}

func (f *fakeAnalyzer) Detect(ctx context.Context, request DetectRequest) (DetectionCandidate, error) {
	if f.detect != nil {
		return f.detect(ctx, request)
	}
	return DetectionCandidate{
		AnalyzerID: f.manifest.ID,
		Confidence: 1,
		Reason:     "test marker",
	}, nil
}

func (f *fakeAnalyzer) Analyze(ctx context.Context, request AnalyzeRequest) (AnalysisResult, error) {
	if f.analyze != nil {
		return f.analyze(ctx, request)
	}
	return validResult(f.manifest), nil
}

func validManifest(id, language string) Manifest {
	return Manifest{
		ID:         id,
		Version:    "1.0.0",
		Language:   language,
		APIVersion: AnalyzerAPIVersion,
		DetectionMarkers: []DetectionMarker{{
			Kind:   "file",
			Value:  language + ".marker",
			Weight: 1,
		}},
		Capabilities: []string{"detect"},
		Options: []OptionDescriptor{{
			Name:    "include_tests",
			Type:    "boolean",
			Default: false,
		}},
	}
}

func validResult(manifest Manifest) AnalysisResult {
	return AnalysisResult{
		RunID:  "run-test",
		Status: StatusComplete,
		Analyzer: AnalyzerInfo{
			ID:         manifest.ID,
			Version:    manifest.Version,
			Language:   manifest.Language,
			APIVersion: manifest.APIVersion,
		},
		Project: ProjectInfo{
			RootLabel: "fixture",
			Boundary:  "test",
		},
	}
}

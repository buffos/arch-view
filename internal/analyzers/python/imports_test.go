package pyanalyzer

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model/canonical"
)

func TestAnalyzeResolvesPythonImportsAndPreservesUncertainty(t *testing.T) {
	root := t.TempDir()
	writePythonFixture(t, filepath.Join(root, "pyproject.toml"), "[tool.setuptools.packages.find]\nwhere = ['src']\n")
	writePythonFixture(t, filepath.Join(root, "src", "app", "__init__.py"), "from .sub.service import Service\nfrom . import models\n")
	writePythonFixture(t, filepath.Join(root, "src", "app", "sub", "service.py"), "import app.models\nfrom . import models\nfrom ..shared import helper\nimport os\nimport requests\nimport app.missing\nimport importlib\n\nif TYPE_CHECKING:\n    from app import models\n\nloaded = importlib.import_module(module_name)\nother = __import__('app.models')\n")
	writePythonFixture(t, filepath.Join(root, "src", "app", "models.py"), "MODEL = True\n")
	writePythonFixture(t, filepath.Join(root, "src", "app", "shared", "helper.py"), "HELPER = True\n")
	writePythonFixture(t, filepath.Join(root, "src", "consumer.py"), "from app import Service\nfrom app import models\nfrom app.sub.service import Service\nfrom app import missing_name\n")
	writePythonFixture(t, filepath.Join(root, "src", "side_effect.py"), "open('must-not-be-created', 'w').write('executed')\n")

	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: pythonOptions(t, nil)})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if result.Status != analysis.StatusPartial {
		t.Fatalf("status = %q, want partial; diagnostics=%#v", result.Status, result.Diagnostics)
	}
	result.RunID = "test-run"
	if err := analysis.ValidateAnalysisResult(result, New().Manifest(), root); err != nil {
		t.Fatalf("common analysis validation: %v", err)
	}
	model, err := canonical.Normalize(result)
	if err != nil {
		t.Fatalf("canonical model normalization: %v", err)
	}
	if len(model.Relationships) == 0 || len(model.SourceReferences) == 0 {
		t.Fatalf("canonical model lost Python import observations: %#v", model)
	}
	if _, err := os.Stat(filepath.Join(root, "must-not-be-created")); !os.IsNotExist(err) {
		t.Fatalf("analysis executed target code: %v", err)
	}

	serviceID := "py:module:app.sub.service"
	modelsID := "py:module:app.models"
	helperID := "py:module:app.shared.helper"
	if relation := findPythonRelationship(result.Relationships, serviceID, modelsID); relation == nil || len(relation.SourceReferenceIDs) != 2 {
		t.Fatalf("service -> models relationship did not aggregate import evidence: %#v", relation)
	}
	if relation := findPythonRelationship(result.Relationships, serviceID, helperID); relation == nil || relation.Metadata["relative"] != true {
		t.Fatalf("relative parent import was not resolved: %#v", relation)
	}
	if relation := findPythonRelationship(result.Relationships, "py:module:consumer", serviceID); relation == nil || relation.Metadata["reexport"] != true || relation.Metadata["resolution_kinds"] != nil && !containsString(relation.Metadata["resolution_kinds"].([]string), "reexport") {
		t.Fatalf("package re-export was not resolved: %#v", relation)
	}
	if relation := findPythonRelationship(result.Relationships, "py:module:consumer", serviceID); relation == nil || !containsString(relation.Metadata["resolution_kinds"].([]string), "base") {
		t.Fatalf("local module symbol import was not retained: %#v", relation)
	}
	for _, relation := range result.Relationships {
		if relation.ToModuleID == "" && relation.ToReferenceID == "" {
			t.Fatalf("unexpected empty target relationship: %#v", relation)
		}
	}
	if reference := findPythonReference(result.References, "os", "standard_library"); reference == nil {
		t.Fatalf("standard-library reference missing: %#v", result.References)
	}
	if reference := findPythonReference(result.References, "requests", "external"); reference == nil {
		t.Fatalf("third-party reference missing: %#v", result.References)
	}
	if reference := findPythonReference(result.References, "app.missing", "unresolved"); reference == nil {
		t.Fatalf("unresolved local-looking reference missing: %#v", result.References)
	}
	if reference := findPythonReference(result.References, "app.missing_name", "unresolved"); reference == nil {
		t.Fatalf("unresolved package attribute reference missing: %#v", result.References)
	}
	dynamicReference := findPythonReference(result.References, "app.models", "dynamic")
	if dynamicReference == nil {
		t.Fatalf("dynamic literal reference missing: %#v", result.References)
	}
	dynamicRelationship := findPythonReferenceRelationship(result.Relationships, serviceID, dynamicReference.ID)
	if dynamicRelationship == nil || dynamicRelationship.Confidence == nil || dynamicRelationship.Confidence.Basis != "dynamic" || dynamicRelationship.Confidence.Score != 0.2 {
		t.Fatalf("dynamic relationship confidence = %#v", dynamicRelationship)
	}
	if !hasDiagnostic(result.Diagnostics, "python_dynamic_import") || !hasDiagnostic(result.Diagnostics, "python_unresolved_import") || !hasDiagnostic(result.Diagnostics, "python_conditional_import") {
		t.Fatalf("uncertainty diagnostics missing: %#v", result.Diagnostics)
	}

	repeat, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: pythonOptions(t, nil)})
	if err != nil {
		t.Fatalf("repeat analyze: %v", err)
	}
	repeat.RunID = "test-run"
	firstJSON, _ := json.Marshal(result)
	repeatJSON, _ := json.Marshal(repeat)
	if string(firstJSON) != string(repeatJSON) {
		t.Fatalf("analysis is not byte-stable:\n%s\n%s", firstJSON, repeatJSON)
	}
	repeatModel, err := canonical.Normalize(repeat)
	if err != nil {
		t.Fatalf("normalize repeated analysis: %v", err)
	}
	modelJSON, _ := json.Marshal(model)
	repeatModelJSON, _ := json.Marshal(repeatModel)
	if string(modelJSON) != string(repeatModelJSON) {
		t.Fatalf("canonical model is not byte-stable:\n%s\n%s", modelJSON, repeatModelJSON)
	}
}

func TestAnalyzeClassifiesPythonVersionedStandardLibraryStatically(t *testing.T) {
	root := t.TempDir()
	writePythonFixture(t, filepath.Join(root, "pyproject.toml"), "[project]\nrequires-python = '>=3.10'\n")
	writePythonFixture(t, filepath.Join(root, "consumer.py"), "import tomllib\nimport dataclasses\n")

	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: pythonOptions(t, nil)})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if reference := findPythonReference(result.References, "tomllib", "external"); reference == nil {
		t.Fatalf("tomllib should not be classified as stdlib for Python 3.10: %#v", result.References)
	}
	if reference := findPythonReference(result.References, "dataclasses", "standard_library"); reference == nil {
		t.Fatalf("dataclasses should be classified as stdlib for Python 3.10: %#v", result.References)
	}
}

func TestAnalyzeUsesVersionedRulesForRemovedAndCurrentStdlibModules(t *testing.T) {
	root := t.TempDir()
	writePythonFixture(t, filepath.Join(root, "pyproject.toml"), "[project]\nname = 'fixture'\n")
	writePythonFixture(t, filepath.Join(root, "consumer.py"), "import audioop\nimport distutils\nimport lib2to3\nimport nntplib\nimport shlex\n")

	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: pythonOptions(t, map[string]any{"python_version": "3.13"})})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	for _, name := range []string{"audioop", "distutils", "lib2to3", "nntplib"} {
		if reference := findPythonReference(result.References, name, "external"); reference == nil {
			t.Fatalf("removed Python 3.13 module %q was not external: %#v", name, result.References)
		}
	}
	if reference := findPythonReference(result.References, "shlex", "standard_library"); reference == nil {
		t.Fatalf("current standard-library module shlex was misclassified: %#v", result.References)
	}
}

func TestAnalyzeResolvesUnicodePythonModuleNames(t *testing.T) {
	root := t.TempDir()
	writePythonFixture(t, filepath.Join(root, "pyproject.toml"), "[project]\nname = 'fixture'\n")
	writePythonFixture(t, filepath.Join(root, "café.py"), "VALUE = 1\n")
	writePythonFixture(t, filepath.Join(root, "consumer.py"), "import café\n")

	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: pythonOptions(t, nil)})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if relation := findPythonRelationship(result.Relationships, "py:module:consumer", "py:module:café"); relation == nil {
		t.Fatalf("Unicode module import was not resolved: relationships=%#v references=%#v", result.Relationships, result.References)
	}
}

func TestExtractPythonImportsSupportsMultilineAndAliases(t *testing.T) {
	content := "from .service import (\n    Service,\n    Helper as PublicHelper,\n)\nimport app.models, app.views as views\n"
	observations, diagnostics := extractPythonImports("src/app/consumer.py", content, "py:module:app.consumer", false)
	if len(diagnostics) != 0 || len(observations) != 4 {
		t.Fatalf("observations=%#v diagnostics=%#v", observations, diagnostics)
	}
	if observations[0].RelativeLevel != 1 || observations[0].ImportedName != "Service" || observations[0].Source.Start.Line != 1 {
		t.Fatalf("first import = %#v", observations[0])
	}
	if observations[1].Alias != "PublicHelper" || observations[1].ImportedName != "Helper" {
		t.Fatalf("aliased from-import = %#v", observations[1])
	}
	if observations[2].Module != "app.models" || observations[3].Alias != "views" {
		t.Fatalf("absolute imports = %#v", observations[2:])
	}
}

func TestAnalyzePreservesNamespacePackagesAndBlocksAmbiguousParents(t *testing.T) {
	root := t.TempDir()
	writePythonFixture(t, filepath.Join(root, "pyproject.toml"), "[tool.setuptools.packages.find]\nwhere = ['src']\n")
	writePythonFixture(t, filepath.Join(root, "src", "ns", "alpha.py"), "import ns.beta\n")
	writePythonFixture(t, filepath.Join(root, "src", "ns", "beta.py"), "VALUE = True\n")
	writePythonFixture(t, filepath.Join(root, "src", "conflict.py"), "VALUE = True\n")
	writePythonFixture(t, filepath.Join(root, "src", "conflict", "__init__.py"), "VALUE = True\n")
	writePythonFixture(t, filepath.Join(root, "src", "conflict", "child.py"), "VALUE = True\n")
	writePythonFixture(t, filepath.Join(root, "src", "consumer.py"), "import conflict.child\n")

	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: pythonOptions(t, nil)})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if relation := findPythonRelationship(result.Relationships, "py:module:ns.alpha", "py:module:ns.beta"); relation == nil {
		t.Fatalf("namespace package import was not resolved: %#v", result.Relationships)
	}
	if !hasDiagnostic(result.Diagnostics, "python_conflicting_layout") {
		t.Fatalf("ambiguous package/module layout diagnostic missing: %#v", result.Diagnostics)
	}
	if relation := findPythonRelationship(result.Relationships, "py:module:consumer", "py:module:conflict.child"); relation != nil {
		t.Fatalf("ambiguous parent produced a local relationship: %#v", relation)
	}
	if reference := findPythonReference(result.References, "conflict.child", "unresolved"); reference == nil {
		t.Fatalf("ambiguous child reference missing: %#v", result.References)
	}
}

func TestAnalyzeBlocksMixedRegularAndNamespacePackageRoots(t *testing.T) {
	root := t.TempDir()
	writePythonFixture(t, filepath.Join(root, "pyproject.toml"), "[project]\nname = 'fixture'\n")
	writePythonFixture(t, filepath.Join(root, "first", "pkg", "__init__.py"), "VALUE = True\n")
	writePythonFixture(t, filepath.Join(root, "second", "pkg", "child.py"), "VALUE = True\n")
	writePythonFixture(t, filepath.Join(root, "second", "consumer.py"), "import pkg.child\n")

	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{
		ProjectRoot: root,
		Options:     pythonOptions(t, map[string]any{"source_roots": []string{"first", "second"}}),
	})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if !hasDiagnostic(result.Diagnostics, "python_conflicting_layout") {
		t.Fatalf("mixed package-root diagnostic missing: %#v", result.Diagnostics)
	}
	if relation := findPythonRelationship(result.Relationships, "py:module:consumer", "py:module:pkg.child"); relation != nil {
		t.Fatalf("mixed package roots produced a local relationship: %#v", relation)
	}
	if reference := findPythonReference(result.References, "pkg.child", "unresolved"); reference == nil {
		t.Fatalf("mixed package-root unresolved reference missing: %#v", result.References)
	}
}

func TestExtractPythonImportsIgnoresDynamicLookingDefinitionsAndRawStringContents(t *testing.T) {
	content := "def eval(value):\n" +
		"    return value\n\n" +
		"class ExtensionManager(Base):\n" +
		"    pass\n\n" +
		"doc = r\"\"\"text \\\"\"\" not close\n" +
		"import fake.module\n" +
		"\"\"\"\n" +
		"import real.module\n"
	observations, diagnostics := extractPythonImports("consumer.py", content, "py:module:consumer", false)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if len(observations) != 1 || observations[0].Spelling != "real.module" {
		t.Fatalf("raw string or definitions produced false imports: %#v", observations)
	}
}

func TestExtractPythonImportsDetectsDynamicImportAliases(t *testing.T) {
	content := "from importlib import import_module as load\n" +
		"from builtins import __import__ as import_name\n" +
		"load(module_name)\n" +
		"import_name(target_name)\n"
	observations, diagnostics := extractPythonImports("consumer.py", content, "py:module:consumer", false)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	dynamic := make([]pythonImportObservation, 0, 2)
	for _, observation := range observations {
		if observation.Kind == "dynamic" {
			dynamic = append(dynamic, observation)
		}
	}
	if len(dynamic) != 2 || dynamic[0].DynamicFunction != "load" || dynamic[1].DynamicFunction != "import_name" {
		t.Fatalf("dynamic aliases = %#v", dynamic)
	}
}

func TestExtractPythonImportsDoesNotFabricateDynamicCallsFromNamesAlone(t *testing.T) {
	content := "def import_module(value):\n" +
		"    return value\n\n" +
		"import_module('ordinary.value')\n" +
		"manager.ExtensionManager('ordinary.manager')\n" +
		"eval('1 + 1')\n" +
		"exec('value = 1')\n"
	observations, diagnostics := extractPythonImports("consumer.py", content, "py:module:consumer", false)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	for _, observation := range observations {
		if observation.Kind == "dynamic" {
			t.Fatalf("ordinary call was fabricated as a dynamic import: %#v", observation)
		}
	}
}

func TestExtractPythonImportsDetectsQualifiedModuleAliases(t *testing.T) {
	content := "import importlib as loader\n" +
		"import importlib.util as import_util\n" +
		"loader.import_module(module_name)\n" +
		"import_util.find_spec(module_name)\n"
	observations, diagnostics := extractPythonImports("consumer.py", content, "py:module:consumer", false)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	dynamicFunctions := make([]string, 0, 2)
	for _, observation := range observations {
		if observation.Kind == "dynamic" {
			dynamicFunctions = append(dynamicFunctions, observation.DynamicFunction)
		}
	}
	if len(dynamicFunctions) != 2 || dynamicFunctions[0] != "loader.import_module" || dynamicFunctions[1] != "import_util.find_spec" {
		t.Fatalf("qualified dynamic aliases = %#v; observations=%#v", dynamicFunctions, observations)
	}
}

func TestExtractPythonImportsDoesNotInventPartialDynamicLiteralTargets(t *testing.T) {
	content := "importlib.import_module(\"app\" + suffix)\n" +
		"importlib.import_module(\"app.\" \"models\")\n"
	observations, diagnostics := extractPythonImports("consumer.py", content, "py:module:consumer", false)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	dynamic := make([]pythonImportObservation, 0, 2)
	for _, observation := range observations {
		if observation.Kind == "dynamic" {
			dynamic = append(dynamic, observation)
		}
	}
	if len(dynamic) != 2 || dynamic[0].DynamicTarget != "" || dynamic[1].DynamicTarget != "app.models" {
		t.Fatalf("dynamic targets = %#v", dynamic)
	}
}

func TestAnalyzeSuppressesPythonSelfDependencyEdges(t *testing.T) {
	root := t.TempDir()
	writePythonFixture(t, filepath.Join(root, "pyproject.toml"), "[project]\nname = 'fixture'\n")
	writePythonFixture(t, filepath.Join(root, "self_module.py"), "import self_module\n")
	writePythonFixture(t, filepath.Join(root, "package", "__init__.py"), "from . import *\n")

	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: pythonOptions(t, nil)})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	for _, relationship := range result.Relationships {
		if relationship.ToModuleID != "" && relationship.FromModuleID == relationship.ToModuleID {
			t.Fatalf("self import produced a graph self-edge: %#v", relationship)
		}
	}
}

func TestStaticPythonStringPreservesEscapedDoubleQuotes(t *testing.T) {
	value, ok := staticPythonString("\"app.\\\"models\"")
	if !ok || value != "app.\"models" {
		t.Fatalf("decoded Python string = %q, ok=%v", value, ok)
	}
}

func TestEvaluatePythonConditionRequiresAnExactVersionPredicate(t *testing.T) {
	if state := evaluatePythonCondition("if feature_flag and sys.version_info >= (3, 11):", "3.12"); state != pythonConditionUnknown {
		t.Fatalf("compound condition state = %v, want unknown", state)
	}
	if state := evaluatePythonCondition("if SYS.version_info >= (3, 11):", "3.12"); state != pythonConditionUnknown {
		t.Fatalf("mis-cased condition state = %v, want unknown", state)
	}
	if state := evaluatePythonCondition("if sys.version_info >= (3, 11):", "3.12"); state != pythonConditionTrue {
		t.Fatalf("exact version condition state = %v, want true", state)
	}
}

func TestConditionalPythonReexportsRemainInferred(t *testing.T) {
	root := t.TempDir()
	writePythonFixture(t, filepath.Join(root, "pyproject.toml"), "[tool.setuptools.packages.find]\nwhere = ['src']\n")
	writePythonFixture(t, filepath.Join(root, "src", "conditional", "__init__.py"), "if TYPE_CHECKING:\n    from .impl import Service\n")
	writePythonFixture(t, filepath.Join(root, "src", "conditional", "impl.py"), "class Service:\n    pass\n")
	writePythonFixture(t, filepath.Join(root, "src", "consumer.py"), "from conditional import Service\n")

	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: pythonOptions(t, nil)})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	relation := findPythonRelationship(result.Relationships, "py:module:consumer", "py:module:conditional.impl")
	if relation == nil {
		t.Fatalf("conditional re-export relationship missing: %#v", result.Relationships)
	}
	if relation.Confidence == nil || relation.Confidence.Score != 0.5 || relation.Confidence.Basis != "inferred" {
		t.Fatalf("conditional re-export confidence = %#v", relation.Confidence)
	}
	if relation.Metadata["conditional"] != true {
		t.Fatalf("conditional re-export metadata = %#v", relation.Metadata)
	}
	conditions, ok := relation.Metadata["conditions"].([]string)
	if !ok || !containsString(conditions, "if TYPE_CHECKING:") {
		t.Fatalf("conditional re-export conditions = %#v", relation.Metadata)
	}
}

func TestMultilineConditionalPythonImportsRemainInferred(t *testing.T) {
	root := t.TempDir()
	writePythonFixture(t, filepath.Join(root, "pyproject.toml"), "[project]\nname = 'fixture'\n")
	writePythonFixture(t, filepath.Join(root, "target.py"), "VALUE = 1\n")
	writePythonFixture(t, filepath.Join(root, "consumer.py"), "if (\n    TYPE_CHECKING\n):\n    import target\n")

	result, err := New().Analyze(context.Background(), analysis.AnalyzeRequest{ProjectRoot: root, Options: pythonOptions(t, nil)})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	relation := findPythonRelationship(result.Relationships, "py:module:consumer", "py:module:target")
	if relation == nil {
		t.Fatalf("multiline conditional import relationship missing: %#v", result.Relationships)
	}
	if relation.Confidence == nil || relation.Confidence.Basis != "inferred" || relation.Confidence.Score != 0.5 {
		t.Fatalf("multiline conditional confidence = %#v", relation.Confidence)
	}
	if !hasDiagnostic(result.Diagnostics, "python_conditional_import") {
		t.Fatalf("multiline conditional diagnostic missing: %#v", result.Diagnostics)
	}
}

func findPythonRelationship(values []analysis.RelationshipObservation, from, target string) *analysis.RelationshipObservation {
	for index := range values {
		value := &values[index]
		if value.FromModuleID != from {
			continue
		}
		if target != "" && value.ToModuleID == target {
			return value
		}
	}
	return nil
}

func findPythonReference(values []analysis.Reference, name, scope string) *analysis.Reference {
	for index := range values {
		value := &values[index]
		if value.Name == name && value.Scope == scope {
			return value
		}
	}
	return nil
}

func findPythonReferenceRelationship(values []analysis.RelationshipObservation, from, referenceID string) *analysis.RelationshipObservation {
	for index := range values {
		value := &values[index]
		if value.FromModuleID == from && value.ToReferenceID == referenceID {
			return value
		}
	}
	return nil
}

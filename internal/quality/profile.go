package quality

import (
	"fmt"
	"math"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

type ErrorCode string

const (
	ErrorCatalogInvalid    ErrorCode = "QualityCatalogInvalid"
	ErrorCatalogConflict   ErrorCode = "QualityCatalogConflict"
	ErrorProfileInvalid    ErrorCode = "QualityProfileInvalid"
	ErrorRuleUnknown       ErrorCode = "QualityRuleUnknown"
	ErrorRuleUnsupported   ErrorCode = "QualityRuleUnsupported"
	ErrorParameterInvalid  ErrorCode = "QualityParameterInvalid"
	ErrorConstraintInvalid ErrorCode = "QualityConstraintInvalid"
	ErrorMetricUnavailable ErrorCode = "MetricInputUnavailable"
	ErrorEvaluationFailed  ErrorCode = "QualityRuleEvaluationFailed"
	ErrorReportInvalid     ErrorCode = "QualityReportInvalid"
)

type QualityError struct {
	Code        ErrorCode
	Message     string
	Details     map[string]any
	Diagnostics []QualityDiagnostic
}

func (err *QualityError) Error() string {
	if err == nil {
		return ""
	}
	if err.Code == "" {
		return err.Message
	}
	return string(err.Code) + ": " + err.Message
}

func newQualityError(code ErrorCode, message string, details map[string]any) *QualityError {
	return &QualityError{Code: code, Message: message, Details: details}
}

// ProfileValidationError retains machine-readable diagnostics while allowing
// callers to handle validation through the ordinary error interface.
type ProfileValidationError struct {
	Diagnostics []QualityDiagnostic
}

func (err *ProfileValidationError) Error() string {
	if err == nil || len(err.Diagnostics) == 0 {
		return "quality profile is invalid"
	}
	return fmt.Sprintf("quality profile is invalid (%d diagnostic(s))", len(err.Diagnostics))
}

func (err *ProfileValidationError) Unwrap() error {
	return newQualityError(ErrorProfileInvalid, err.Error(), nil)
}

// ValidateQualityProfile resolves every binding against catalog strategies and
// validates typed rule configuration before any source/model fact is read.
func ValidateQualityProfile(profile QualityProfile, catalog *Catalog) (QualityProfile, error) {
	validated := cloneProfile(profile)
	diagnostics := make([]QualityDiagnostic, 0)
	if validated.SchemaVersion != SchemaVersion {
		diagnostics = append(diagnostics, profileDiagnostic("quality:schema-version", "schema_version", "quality profile schema version is unsupported", map[string]any{"schema_version": validated.SchemaVersion}))
	}
	if !validProfileID(validated.ProfileID) {
		diagnostics = append(diagnostics, profileDiagnostic("quality:profile-id", "profile_id", "profile identity must be a namespaced value such as profile:default", map[string]any{"profile_id": validated.ProfileID}))
	}
	if !validVersion(validated.ProfileVersion) {
		diagnostics = append(diagnostics, profileDiagnostic("quality:profile-version", "profile_version", "profile version must be a stable version value", map[string]any{"profile_version": validated.ProfileVersion}))
	}
	if catalog == nil {
		diagnostics = append(diagnostics, profileDiagnostic("quality:catalog-unavailable", "", "quality rule catalog is unavailable", nil))
	}
	if validated.SeverityPolicy.Namespace == "" && validated.SeverityPolicy.SchemaVersion == "" && validated.SeverityPolicy.Payload == nil {
		validated.SeverityPolicy = TypedConfigBlock{Namespace: "severity:default", SchemaVersion: "1.0.0", Payload: map[string]any{}}
	} else if err := validateTypedBlock(validated.SeverityPolicy, ParameterSchema{Namespace: "severity:default", SchemaVersion: "1.0.0", Fields: map[string]ParameterField{}, AllowAdditional: true}, "severity_policy"); err != nil {
		diagnostics = append(diagnostics, diagnosticFromError(err, "severity_policy"))
	}

	seenRules := make(map[string]struct{}, len(validated.EnabledRules))
	for index := range validated.EnabledRules {
		binding := &validated.EnabledRules[index]
		path := fmt.Sprintf("enabled_rules[%d]", index)
		key := strings.TrimSpace(binding.RuleID) + "\x00" + strings.TrimSpace(binding.RuleVersion)
		if _, exists := seenRules[key]; exists {
			diagnostics = append(diagnostics, profileDiagnostic("quality:duplicate-rule", path, "quality profile cannot bind the same rule version more than once", map[string]any{"rule_id": binding.RuleID, "rule_version": binding.RuleVersion}))
			continue
		}
		seenRules[key] = struct{}{}
		if !validNamespacedID(binding.RuleID) || !validVersion(binding.RuleVersion) {
			diagnostics = append(diagnostics, profileDiagnostic("quality:rule-identity", path, "rule binding requires a safe namespaced ID and version", map[string]any{"rule_id": binding.RuleID, "rule_version": binding.RuleVersion}))
			continue
		}
		var rule QualityRule
		var ok bool
		if catalog != nil {
			rule, ok = catalog.ResolveQualityRule(binding.RuleID, binding.RuleVersion)
		}
		if !ok {
			if catalog != nil && catalog.hasRuleID(binding.RuleID) {
				diagnostics = append(diagnostics, profileDiagnostic(string(ErrorRuleUnsupported), path, "the requested rule version is not registered", map[string]any{"rule_id": binding.RuleID, "rule_version": binding.RuleVersion}))
			} else {
				diagnostics = append(diagnostics, profileDiagnostic(string(ErrorRuleUnknown), path, "the requested quality rule is not registered", map[string]any{"rule_id": binding.RuleID, "rule_version": binding.RuleVersion}))
			}
			continue
		}
		descriptor := descriptorForRule(rule)
		if descriptor.ParameterSchema.Namespace != "" {
			if err := validateTypedBlock(binding.Parameters, descriptor.ParameterSchema, path+".parameters"); err != nil {
				diagnostics = append(diagnostics, diagnosticFromError(err, path+".parameters"))
			}
		}
		if binding.Severity != "" && !validSeverity(binding.Severity) {
			diagnostics = append(diagnostics, profileDiagnostic("quality:severity", path+".severity", "rule severity is invalid", map[string]any{"severity": binding.Severity}))
		}
		if binding.Severity == "" {
			binding.Severity = descriptor.DefaultSeverity
			if binding.Severity == "" {
				binding.Severity = defaultSeverity(descriptor.AssessmentKind)
			}
		}
	}
	couplingPolicy := ""
	for index := range validated.EnabledRules {
		binding := validated.EnabledRules[index]
		if binding.RuleID != "architecture:module.max-efferent-coupling" && binding.RuleID != "architecture:module.max-afferent-coupling" {
			continue
		}
		payload, ok := stringMap(binding.Parameters.Payload)
		if !ok {
			continue
		}
		value, exists := payload["external_policy"]
		if !exists {
			continue
		}
		policy, ok := value.(string)
		if !ok {
			continue
		}
		if couplingPolicy != "" && couplingPolicy != policy {
			diagnostics = append(diagnostics, profileDiagnostic(string(ErrorParameterInvalid), fmt.Sprintf("enabled_rules[%d].parameters.external_policy", index), "efferent and afferent coupling rules must use one shared external/reference policy", map[string]any{"first": couplingPolicy, "second": policy}))
		} else {
			couplingPolicy = policy
		}
	}

	seenConstraints := make(map[string]struct{}, len(validated.Constraints))
	for index := range validated.Constraints {
		constraint := &validated.Constraints[index]
		path := fmt.Sprintf("constraints[%d]", index)
		if !validNamespacedID(constraint.ID) {
			diagnostics = append(diagnostics, profileDiagnostic(string(ErrorConstraintInvalid), path+".id", "architecture constraint requires a namespaced ID", map[string]any{"id": constraint.ID}))
		}
		if _, exists := seenConstraints[constraint.ID]; exists {
			diagnostics = append(diagnostics, profileDiagnostic(string(ErrorConstraintInvalid), path+".id", "architecture constraint IDs must be unique", map[string]any{"id": constraint.ID}))
		}
		seenConstraints[constraint.ID] = struct{}{}
		if constraint.Kind != "forbidden_dependency" && constraint.Kind != "layer_direction" {
			diagnostics = append(diagnostics, profileDiagnostic(string(ErrorConstraintInvalid), path+".kind", "architecture constraint kind is unsupported", map[string]any{"kind": constraint.Kind}))
		}
		if constraint.Parameters.Namespace == "" || constraint.Parameters.SchemaVersion == "" {
			diagnostics = append(diagnostics, profileDiagnostic(string(ErrorConstraintInvalid), path+".parameters", "architecture constraint parameters require a typed namespace and version", nil))
		} else if err := validateArchitectureConstraint(*constraint); err != nil {
			diagnostics = append(diagnostics, diagnosticFromError(err, path+".parameters"))
		}
		if constraint.Provenance.Status == "" {
			constraint.Provenance = FactProvenance{Status: "observed", Basis: "configuration", EvidenceIDs: []string{}, Provider: "quality:profile", ProviderVersion: "1.0.0"}
		}
	}
	if validated.Baseline != nil && !validNamespacedID(validated.Baseline.BaselineID) {
		diagnostics = append(diagnostics, profileDiagnostic(string(ErrorProfileInvalid), "baseline.baseline_id", "baseline reference requires a namespaced ID", map[string]any{"baseline_id": validated.Baseline.BaselineID}))
	}
	if len(diagnostics) > 0 {
		return QualityProfile{}, &ProfileValidationError{Diagnostics: diagnostics}
	}
	sort.Slice(validated.EnabledRules, func(i, j int) bool {
		if validated.EnabledRules[i].RuleID == validated.EnabledRules[j].RuleID {
			return validated.EnabledRules[i].RuleVersion < validated.EnabledRules[j].RuleVersion
		}
		return validated.EnabledRules[i].RuleID < validated.EnabledRules[j].RuleID
	})
	sort.Slice(validated.Constraints, func(i, j int) bool { return validated.Constraints[i].ID < validated.Constraints[j].ID })
	return validated, nil
}

func validateArchitectureConstraint(constraint ArchitectureConstraint) error {
	if !validNamespacedID(constraint.Parameters.Namespace) || !strings.HasPrefix(constraint.Parameters.Namespace, "constraint:") || !validVersion(constraint.Parameters.SchemaVersion) {
		return newQualityError(ErrorConstraintInvalid, "architecture constraint parameters must use a versioned constraint namespace", map[string]any{"namespace": constraint.Parameters.Namespace, "schema_version": constraint.Parameters.SchemaVersion})
	}
	payload, ok := stringMap(constraint.Parameters.Payload)
	if !ok {
		return newQualityError(ErrorConstraintInvalid, "architecture constraint payload must be an object", nil)
	}
	if value, exists := payload["scope_id"]; exists {
		if text, valid := value.(string); !valid || strings.TrimSpace(text) == "" {
			return newQualityError(ErrorConstraintInvalid, "architecture constraint scope_id must be a non-empty string", map[string]any{"scope_id": value})
		}
	}
	switch constraint.Kind {
	case "forbidden_dependency":
		if constraint.Parameters.Namespace != "constraint:forbidden" {
			return newQualityError(ErrorConstraintInvalid, "forbidden-dependency parameters use the wrong typed namespace", map[string]any{"namespace": constraint.Parameters.Namespace})
		}
		if err := validateExplicitSelector(payload, "from"); err != nil {
			return err
		}
		if err := validateExplicitSelector(payload, "to"); err != nil {
			return err
		}
	case "layer_direction":
		if constraint.Parameters.Namespace != "constraint:layers" {
			return newQualityError(ErrorConstraintInvalid, "layer-direction parameters use the wrong typed namespace", map[string]any{"namespace": constraint.Parameters.Namespace})
		}
		direction, directionOK := payload["direction"].(string)
		if directionOK && direction != "lower_to_higher" && direction != "higher_to_lower" && direction != "upward" && direction != "downward" && direction != "same" {
			return newQualityError(ErrorConstraintInvalid, "layer direction is unsupported", map[string]any{"direction": direction})
		}
		fromLayer, fromPresent := payload["from_layer"]
		toLayer, toPresent := payload["to_layer"]
		if fromPresent {
			if _, valid := integerValue(fromLayer); !valid {
				return newQualityError(ErrorConstraintInvalid, "from_layer must be an integer", map[string]any{"from_layer": fromLayer})
			}
		}
		if toPresent {
			if _, valid := integerValue(toLayer); !valid {
				return newQualityError(ErrorConstraintInvalid, "to_layer must be an integer", map[string]any{"to_layer": toLayer})
			}
		}
		if !directionOK && (!fromPresent || !toPresent) {
			return newQualityError(ErrorConstraintInvalid, "layer direction requires direction or both explicit from_layer and to_layer selectors", nil)
		}
		if hasSelectorPrefix(payload, "from") {
			if err := validateExplicitSelector(payload, "from"); err != nil {
				return err
			}
		}
		if hasSelectorPrefix(payload, "to") {
			if err := validateExplicitSelector(payload, "to"); err != nil {
				return err
			}
		}
	default:
		return newQualityError(ErrorConstraintInvalid, "architecture constraint kind is unsupported", map[string]any{"kind": constraint.Kind})
	}
	if err := validateConstraintFields(constraint.Kind, payload); err != nil {
		return err
	}
	return nil
}

func validateConstraintFields(kind string, payload map[string]any) error {
	allowed := map[string]struct{}{"scope_id": {}}
	addSelectorFields := func(prefix string) {
		for _, suffix := range []string{"_module_ids", "_ids", "_tags", "_tag", "_module_patterns", "_stable_key_globs", "_patterns", "_layers", "_layer"} {
			allowed[prefix+suffix] = struct{}{}
		}
	}
	addSelectorFields("from")
	addSelectorFields("to")
	if kind == "layer_direction" {
		allowed["direction"] = struct{}{}
	}
	for field := range payload {
		if _, ok := allowed[field]; !ok {
			return newQualityError(ErrorConstraintInvalid, "architecture constraint contains an unknown parameter", map[string]any{"field": field})
		}
	}
	return nil
}

func validateExplicitSelector(payload map[string]any, prefix string) error {
	if !hasSelectorPrefix(payload, prefix) {
		return newQualityError(ErrorConstraintInvalid, "architecture constraint requires explicit module selectors", map[string]any{"selector": prefix})
	}
	for _, key := range []string{prefix + "_module_ids", prefix + "_ids", prefix + "_tags", prefix + "_tag"} {
		if value, exists := payload[key]; exists {
			values, valid := stringSlice(value)
			if !valid || len(values) == 0 {
				if text, single := value.(string); single && strings.TrimSpace(text) != "" {
					values = []string{text}
					valid = true
				}
			}
			if !valid || len(values) == 0 {
				return newQualityError(ErrorConstraintInvalid, "architecture constraint selector values must be non-empty strings", map[string]any{"field": key})
			}
			for _, item := range values {
				if strings.TrimSpace(item) == "" {
					return newQualityError(ErrorConstraintInvalid, "architecture constraint selector values must be non-empty strings", map[string]any{"field": key})
				}
			}
		}
	}
	for _, key := range []string{prefix + "_module_patterns", prefix + "_stable_key_globs", prefix + "_patterns"} {
		if value, exists := payload[key]; exists {
			values, valid := stringSlice(value)
			if !valid {
				if text, single := value.(string); single {
					values = []string{text}
					valid = true
				}
			}
			if !valid || len(values) == 0 {
				return newQualityError(ErrorConstraintInvalid, "architecture constraint pattern selectors must be non-empty strings", map[string]any{"field": key})
			}
			for _, pattern := range values {
				if strings.TrimSpace(pattern) == "" {
					return newQualityError(ErrorConstraintInvalid, "architecture constraint pattern selectors must be non-empty strings", map[string]any{"field": key})
				}
				if _, err := path.Match(pattern, ""); err != nil {
					return newQualityError(ErrorConstraintInvalid, "architecture constraint pattern selector is invalid", map[string]any{"field": key, "pattern": pattern})
				}
			}
		}
	}
	for _, key := range []string{prefix + "_layers", prefix + "_layer"} {
		if value, exists := payload[key]; exists {
			if _, valid := integerValue(value); valid {
				continue
			}
			values, valid := integerSlice(value)
			if !valid || len(values) == 0 {
				return newQualityError(ErrorConstraintInvalid, "architecture constraint layer selectors must be integers", map[string]any{"field": key})
			}
		}
	}
	return nil
}

func hasSelectorPrefix(payload map[string]any, prefix string) bool {
	for _, key := range []string{prefix + "_module_ids", prefix + "_ids", prefix + "_tags", prefix + "_tag", prefix + "_module_patterns", prefix + "_stable_key_globs", prefix + "_patterns", prefix + "_layers", prefix + "_layer"} {
		if value, exists := payload[key]; exists {
			if values, ok := stringSlice(value); ok && len(values) > 0 {
				return true
			}
			if _, ok := value.(string); ok {
				return true
			}
			if _, ok := integerValue(value); ok {
				return true
			}
			if values, ok := integerSlice(value); ok && len(values) > 0 {
				return true
			}
		}
	}
	return false
}

func (catalog *Catalog) ValidateQualityProfile(profile QualityProfile) (QualityProfile, error) {
	return ValidateQualityProfile(profile, catalog)
}

func profileDiagnostic(code, field, message string, details map[string]any) QualityDiagnostic {
	return QualityDiagnostic{Code: code, Field: field, Message: message, Severity: SeverityError, Details: details}
}

func diagnosticFromError(err error, field string) QualityDiagnostic {
	if qualityErr, ok := err.(*QualityError); ok {
		return QualityDiagnostic{Code: string(qualityErr.Code), Field: field, Message: qualityErr.Message, Severity: SeverityError, Details: qualityErr.Details}
	}
	return QualityDiagnostic{Code: string(ErrorParameterInvalid), Field: field, Message: err.Error(), Severity: SeverityError}
}

func validateTypedBlock(block TypedConfigBlock, schema ParameterSchema, field string) error {
	if strings.TrimSpace(block.Namespace) == "" || strings.TrimSpace(block.SchemaVersion) == "" {
		return newQualityError(ErrorParameterInvalid, "typed configuration requires a namespace and schema version", map[string]any{"field": field})
	}
	if block.Namespace != schema.Namespace || block.SchemaVersion != schema.SchemaVersion {
		return newQualityError(ErrorParameterInvalid, "typed configuration namespace or schema version does not match the registered schema", map[string]any{"expected_namespace": schema.Namespace, "actual_namespace": block.Namespace, "expected_schema_version": schema.SchemaVersion, "actual_schema_version": block.SchemaVersion})
	}
	payload, ok := stringMap(block.Payload)
	if !ok {
		return newQualityError(ErrorParameterInvalid, "typed configuration payload must be an object", map[string]any{"field": field})
	}
	for name, spec := range schema.Fields {
		value, exists := payload[name]
		if !exists {
			if spec.Required {
				return newQualityError(ErrorParameterInvalid, "typed configuration is missing a required field", map[string]any{"field": name})
			}
			continue
		}
		if err := validateParameterValue(name, value, spec); err != nil {
			return err
		}
	}
	if !schema.AllowAdditional {
		for name := range payload {
			if _, exists := schema.Fields[name]; !exists {
				return newQualityError(ErrorParameterInvalid, "typed configuration contains an unknown field", map[string]any{"field": name})
			}
		}
	}
	return nil
}

func validateParameterValue(name string, value any, spec ParameterField) error {
	valid := false
	switch spec.Kind {
	case "string":
		_, valid = value.(string)
	case "integer":
		integer, ok := integerValue(value)
		valid = ok
		if valid && spec.Minimum != nil {
			valid = float64(integer) >= *spec.Minimum
		}
		if valid && spec.Maximum != nil {
			valid = float64(integer) <= *spec.Maximum
		}
	case "decimal":
		decimal, ok := decimalValue(value)
		valid = ok
		if valid && spec.Minimum != nil {
			valid = decimal >= *spec.Minimum
		}
		if valid && spec.Maximum != nil {
			valid = decimal <= *spec.Maximum
		}
	case "boolean":
		_, valid = value.(bool)
	case "string_list":
		values, ok := stringSlice(value)
		valid = ok
		if valid {
			for _, item := range values {
				if strings.TrimSpace(item) == "" {
					valid = false
					break
				}
			}
		}
	default:
		valid = false
	}
	if !valid {
		return newQualityError(ErrorParameterInvalid, "typed configuration field has the wrong type or range", map[string]any{"field": name, "kind": spec.Kind})
	}
	if len(spec.AllowedValues) > 0 {
		text, ok := value.(string)
		if !ok {
			return newQualityError(ErrorParameterInvalid, "allowed values apply only to string fields", map[string]any{"field": name})
		}
		allowed := false
		for _, candidate := range spec.AllowedValues {
			if text == candidate {
				allowed = true
				break
			}
		}
		if !allowed {
			return newQualityError(ErrorParameterInvalid, "typed configuration field has an unsupported value", map[string]any{"field": name, "value": text, "allowed_values": spec.AllowedValues})
		}
	}
	return nil
}

func stringMap(value any) (map[string]any, bool) {
	if value == nil {
		return nil, false
	}
	if result, ok := value.(map[string]any); ok {
		return result, true
	}
	reflected := reflect.ValueOf(value)
	if reflected.Kind() != reflect.Map || reflected.Type().Key().Kind() != reflect.String {
		return nil, false
	}
	result := make(map[string]any, reflected.Len())
	iterator := reflected.MapRange()
	for iterator.Next() {
		result[iterator.Key().String()] = iterator.Value().Interface()
	}
	return result, true
}

func stringSlice(value any) ([]string, bool) {
	reflected := reflect.ValueOf(value)
	if !reflected.IsValid() || (reflected.Kind() != reflect.Slice && reflected.Kind() != reflect.Array) {
		return nil, false
	}
	result := make([]string, reflected.Len())
	for index := 0; index < reflected.Len(); index++ {
		item := reflected.Index(index)
		for item.IsValid() && item.Kind() == reflect.Interface {
			if item.IsNil() {
				return nil, false
			}
			item = item.Elem()
		}
		if !item.IsValid() || item.Kind() != reflect.String {
			return nil, false
		}
		text, ok := item.Interface().(string)
		if !ok {
			return nil, false
		}
		result[index] = text
	}
	return result, true
}

func integerSlice(value any) ([]int64, bool) {
	reflected := reflect.ValueOf(value)
	if !reflected.IsValid() || (reflected.Kind() != reflect.Slice && reflected.Kind() != reflect.Array) {
		return nil, false
	}
	result := make([]int64, reflected.Len())
	for index := 0; index < reflected.Len(); index++ {
		item := reflected.Index(index)
		for item.IsValid() && item.Kind() == reflect.Interface {
			if item.IsNil() {
				return nil, false
			}
			item = item.Elem()
		}
		if !item.IsValid() {
			return nil, false
		}
		if integer, ok := integerValue(item.Interface()); ok {
			result[index] = integer
			continue
		}
		if item.Kind() == reflect.String {
			integer, err := strconv.ParseInt(strings.TrimSpace(item.String()), 10, 64)
			if err == nil {
				result[index] = integer
				continue
			}
		}
		return nil, false
	}
	return result, true
}

func integerValue(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int8:
		return int64(typed), true
	case int16:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case uint:
		if uint64(typed) > uint64(^uint64(0)>>1) {
			return 0, false
		}
		return int64(typed), true
	case uint8:
		return int64(typed), true
	case uint16:
		return int64(typed), true
	case uint32:
		return int64(typed), true
	case uint64:
		if typed > uint64(^uint64(0)>>1) {
			return 0, false
		}
		return int64(typed), true
	case float32:
		if math.IsNaN(float64(typed)) || math.IsInf(float64(typed), 0) || math.Trunc(float64(typed)) != float64(typed) {
			return 0, false
		}
		return int64(typed), true
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) || math.Trunc(typed) != typed || typed > float64(^uint64(0)>>1) || typed < -float64(^uint64(0)>>1)-1 {
			return 0, false
		}
		return int64(typed), true
	default:
		return 0, false
	}
}

func decimalValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case float32:
		return float64(typed), !math.IsNaN(float64(typed)) && !math.IsInf(float64(typed), 0)
	case float64:
		return typed, !math.IsNaN(typed) && !math.IsInf(typed, 0)
	default:
		integer, ok := integerValue(value)
		return float64(integer), ok
	}
}

func cloneProfile(profile QualityProfile) QualityProfile {
	profile.EnabledRules = append([]RuleBinding(nil), profile.EnabledRules...)
	profile.Constraints = append([]ArchitectureConstraint(nil), profile.Constraints...)
	profile.Extensions = append([]ExtensionBlock(nil), profile.Extensions...)
	if profile.EnabledRules == nil {
		profile.EnabledRules = []RuleBinding{}
	}
	if profile.Constraints == nil {
		profile.Constraints = []ArchitectureConstraint{}
	}
	if profile.Extensions == nil {
		profile.Extensions = []ExtensionBlock{}
	}
	return profile
}

func validNamespacedID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || value != strings.TrimSpace(value) || strings.ContainsAny(value, "\x00\r\n\t ") {
		return false
	}
	separator := strings.IndexByte(value, ':')
	return separator > 0 && separator < len(value)-1
}

func validProfileID(value string) bool {
	return validNamespacedID(value) && strings.HasPrefix(value, "profile:")
}

func validVersion(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || value != strings.TrimSpace(value) || strings.ContainsAny(value, "\x00\r\n\t ") {
		return false
	}
	if value[0] == 'v' {
		value = value[1:]
	}
	if value == "" {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') && character != '.' && character != '-' && character != '+' {
			return false
		}
	}
	return true
}

func validSeverity(value string) bool {
	switch value {
	case SeverityInfo, SeverityWarning, SeverityError, SeverityBlocker:
		return true
	default:
		return false
	}
}

func defaultSeverity(assessment string) string {
	if assessment == AssessmentSignal {
		return SeverityInfo
	}
	return SeverityWarning
}

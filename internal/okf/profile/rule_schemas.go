package profile

func objectSchema(properties map[string]any, required ...string) map[string]any {
	return map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": true}
}

func textSchema() map[string]any { return map[string]any{"type": "string", "pattern": `\S`} }

func presentationProperties() map[string]any {
	properties := map[string]any{"annotations": map[string]any{"type": "object"}}
	for _, key := range []string{"role", "token", "shape", "label", "state"} {
		properties[key] = map[string]any{"type": "string"}
	}
	return properties
}

func (metadataEqualsStrategy) ParameterSchema() map[string]any {
	properties := presentationProperties()
	properties["field"], properties["value"] = textSchema(), map[string]any{}
	return objectSchema(properties, "field", "value")
}

func (metadataContainsStrategy) ParameterSchema() map[string]any {
	properties := presentationProperties()
	properties["field"] = textSchema()
	properties["value"] = map[string]any{"anyOf": []any{textSchema(), map[string]any{"type": []string{"boolean", "number", "object", "array"}}}}
	return objectSchema(properties, "field", "value")
}

func (stateMappingStrategy) ParameterSchema() map[string]any {
	properties := presentationProperties()
	properties["field"] = textSchema()
	properties["mapping"] = map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}}
	return objectSchema(properties, "mapping")
}

func (labelTemplateStrategy) ParameterSchema() map[string]any {
	return objectSchema(map[string]any{"template": textSchema()}, "template")
}

func (visibilityStrategy) ParameterSchema() map[string]any {
	return objectSchema(map[string]any{"field": textSchema(), "value": map[string]any{}, "visible": map[string]any{"type": "boolean", "default": true}}, "field", "value")
}

package domain

import "encoding/json"

type navigationJSON struct {
	Depth         *int `json:"default_depth,omitempty"`
	Nodes         *int `json:"max_nodes,omitempty"`
	Relationships *int `json:"max_relationships,omitempty"`
}

func (value *NavigationSettings) UnmarshalJSON(data []byte) error {
	var raw navigationJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*value = NavigationSettings{DepthConfigured: raw.Depth != nil, NodesConfigured: raw.Nodes != nil, RelationshipsConfigured: raw.Relationships != nil}
	if raw.Depth != nil {
		value.DefaultDepth = *raw.Depth
	}
	if raw.Nodes != nil {
		value.MaxNodes = *raw.Nodes
	}
	if raw.Relationships != nil {
		value.MaxRelationships = *raw.Relationships
	}
	return nil
}

func (value NavigationSettings) MarshalJSON() ([]byte, error) {
	var raw navigationJSON
	if value.DepthConfigured || value.DefaultDepth != 0 {
		raw.Depth = &value.DefaultDepth
	}
	if value.NodesConfigured || value.MaxNodes != 0 {
		raw.Nodes = &value.MaxNodes
	}
	if value.RelationshipsConfigured || value.MaxRelationships != 0 {
		raw.Relationships = &value.MaxRelationships
	}
	return json.Marshal(raw)
}

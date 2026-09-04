package domain

import (
	"encoding/json"
	"fmt"
)

// Profile keeps extension fields opaque so the application can edit known
// presentation settings without destroying profile-specific additions.
type profileExtensions map[string]json.RawMessage

var profileJSONFields = map[string]struct{}{
	"profile_id": {}, "name": {}, "origin": {}, "bases": {}, "rules": {},
	"hierarchy": {}, "relationships": {}, "state": {}, "node_fields": {},
	"navigation": {}, "style": {}, "details": {}, "layout": {},
	"revision": {}, "status": {}, "immutable": {},
}

func (profile *Profile) UnmarshalJSON(data []byte) error {
	if profile == nil {
		return fmt.Errorf("cannot decode a profile into nil")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw == nil {
		return fmt.Errorf("profile must be a JSON object")
	}
	var decoded profileJSONAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*profile = Profile(decoded)
	profile.extensions = nil
	for key, value := range raw {
		if _, known := profileJSONFields[key]; known {
			continue
		}
		if profile.extensions == nil {
			profile.extensions = make(profileExtensions)
		}
		profile.extensions[key] = append(json.RawMessage(nil), value...)
	}
	return nil
}

func (profile Profile) MarshalJSON() ([]byte, error) {
	known, err := json.Marshal(profileJSONAlias(profile))
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(known, &raw); err != nil {
		return nil, err
	}
	for key, value := range profile.extensions {
		if _, known := profileJSONFields[key]; known {
			continue
		}
		raw[key] = append(json.RawMessage(nil), value...)
	}
	if profile.NodeFields != nil {
		raw["node_fields"], err = json.Marshal(profile.NodeFields)
		if err != nil {
			return nil, err
		}
	}
	return json.Marshal(raw)
}

// profileJSONAlias prevents the custom codec from calling itself recursively.
type profileJSONAlias Profile

func cloneProfileExtensions(value profileExtensions) profileExtensions {
	if value == nil {
		return nil
	}
	result := make(profileExtensions, len(value))
	for key, raw := range value {
		result[key] = append(json.RawMessage(nil), raw...)
	}
	return result
}

// MergeProfileExtensions carries opaque top-level fields through inherited
// profile composition. Known fields are still owned by the typed model.
func MergeProfileExtensions(base, overlay Profile) Profile {
	if base.extensions == nil && overlay.extensions == nil {
		return base
	}
	if base.extensions == nil {
		base.extensions = make(profileExtensions)
	}
	for key, value := range overlay.extensions {
		base.extensions[key] = append(json.RawMessage(nil), value...)
	}
	return base
}

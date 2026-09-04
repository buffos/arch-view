package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestProfileJSONRoundTripPreservesUnknownTopLevelFields(t *testing.T) {
	var value Profile
	if err := json.Unmarshal([]byte(`{"profile_id":"project:review","name":"Review","layout":{"algorithm":"layered","options":{}},"extension":{"enabled":true},"vendor_note":["keep"]}`), &value); err != nil {
		t.Fatal(err)
	}
	value.Name = "Updated"
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, `"extension":{"enabled":true}`) || !strings.Contains(text, `"vendor_note":["keep"]`) {
		t.Fatalf("unknown profile fields were lost: %s", text)
	}
	if !strings.Contains(text, `"name":"Updated"`) {
		t.Fatalf("known profile field was not updated: %s", text)
	}
}

func TestCloneProfilePreservesUnknownTopLevelFields(t *testing.T) {
	var value Profile
	if err := json.Unmarshal([]byte(`{"profile_id":"project:review","extension":{"value":"opaque"}}`), &value); err != nil {
		t.Fatal(err)
	}
	clone := CloneProfile(value)
	data, err := json.Marshal(clone)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"extension":{"value":"opaque"}`) {
		t.Fatalf("clone lost opaque extension: %s", data)
	}
}

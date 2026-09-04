// Package projectdocument owns atomic read-modify-write transactions for the
// shared .archview.json document. Section owners validate their own values.
package projectdocument

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

const MaxBytes = 4 << 20

var writes sync.Mutex

func Object(data []byte) (map[string]json.RawMessage, error) {
	var value map[string]json.RawMessage
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	if value == nil {
		return nil, fmt.Errorf("project configuration must be a JSON object")
	}
	return value, nil
}

func Read(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, MaxBytes+1))
	if err == nil && len(data) > MaxBytes {
		err = fmt.Errorf("project configuration exceeds %d bytes", MaxBytes)
	}
	return data, err
}

// Update re-reads the destination while holding the common writer lock. Initial
// is used only for a missing file; nil requires the destination to exist.
func Update(path string, initial []byte, transform func([]byte) ([]byte, error)) ([]byte, error) {
	writes.Lock()
	defer writes.Unlock()
	data, err := Read(path)
	if os.IsNotExist(err) && initial != nil {
		data, err = initial, nil
	}
	if err != nil {
		return nil, err
	}
	if _, err := Object(data); err != nil {
		return nil, err
	}
	updated, err := transform(data)
	if err != nil {
		return nil, err
	}
	if len(updated) > MaxBytes {
		return nil, fmt.Errorf("project configuration exceeds %d bytes", MaxBytes)
	}
	if _, err := Object(updated); err != nil {
		return nil, err
	}
	if err := writeAtomically(path, updated); err != nil {
		return nil, err
	}
	return updated, nil
}

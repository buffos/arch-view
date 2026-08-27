package viewer

import "fmt"

// Asset returns one of the browser assets embedded in the viewer. Exporters
// use the same presentation shell while replacing network-backed loading with
// an embedded model, scene catalog, layout profile, and pinned ELK runtime.
func Asset(name string) ([]byte, error) {
	data, err := webFiles.ReadFile("web/" + name)
	if err != nil {
		return nil, fmt.Errorf("viewer asset %q: %w", name, err)
	}
	return data, nil
}

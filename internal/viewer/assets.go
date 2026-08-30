package viewer

import (
	"fmt"
	"strings"
)

var stylesheetModules = []string{
	"00-tokens.css",
	"01-foundation.css",
	"02-typography.css",
	"03-layout.css",
	"04-controls.css",
	"05-components.css",
	"06-graph.css",
	"07-details.css",
	"08-support.css",
	"09-modes.css",
	"10-settings.css",
	"11-inspection.css",
}

// Asset returns one of the browser assets embedded in the viewer. Exporters
// use the same presentation shell while replacing network-backed loading with
// an embedded model, scene catalog, layout profile, and pinned ELK runtime.
func Asset(name string) ([]byte, error) {
	if name == "styles.css" {
		return bundledStylesheet()
	}
	data, err := webFiles.ReadFile("web/" + name)
	if err != nil {
		return nil, fmt.Errorf("viewer asset %q: %w", name, err)
	}
	return data, nil
}

func bundledStylesheet() ([]byte, error) {
	var bundle strings.Builder
	for _, name := range stylesheetModules {
		data, err := webFiles.ReadFile("web/styles/" + name)
		if err != nil {
			return nil, fmt.Errorf("viewer stylesheet module %q: %w", name, err)
		}
		bundle.WriteString("/* ")
		bundle.WriteString(name)
		bundle.WriteString(" */\n")
		_, _ = bundle.Write(data)
		if len(data) == 0 || data[len(data)-1] != '\n' {
			bundle.WriteByte('\n')
		}
	}
	return []byte(bundle.String()), nil
}

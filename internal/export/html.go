package export

import (
	"encoding/json"
	"html"
	"strings"

	"github.com/buffo/arch-view/internal/analysis"
	"github.com/buffo/arch-view/internal/model"
	"github.com/buffo/arch-view/internal/viewer"
)

func renderHTML(value model.Model, request Request) ([]byte, map[string]any, error) {
	bundle, provenance, err := sceneCatalog(value, request)
	if err != nil {
		return nil, nil, err
	}
	bundleData, err := json.Marshal(bundle)
	if err != nil {
		return nil, nil, analysis.WrapHostError(analysis.ErrHostFailure, "HTML export data could not be serialized", err, nil)
	}
	indexData, err := viewer.Asset("index.html")
	if err != nil {
		return nil, nil, analysis.WrapHostError(analysis.ErrHostFailure, "HTML export template could not be loaded", err, nil)
	}
	stylesData, err := viewer.Asset("styles.css")
	if err != nil {
		return nil, nil, analysis.WrapHostError(analysis.ErrHostFailure, "HTML export stylesheet could not be loaded", err, nil)
	}
	appData, err := viewer.Asset("app.js")
	if err != nil {
		return nil, nil, analysis.WrapHostError(analysis.ErrHostFailure, "HTML export application could not be loaded", err, nil)
	}

	template := normalizeNewlines(string(indexData))
	styles := normalizeNewlines(string(stylesData))
	app := normalizeNewlines(string(appData))
	template, err = replaceRequired(template, `    <link rel="stylesheet" href="/assets/styles.css">`, "    <style>\n"+styles+"\n    </style>")
	if err != nil {
		return nil, nil, err
	}
	template, err = replaceRequired(template, "    <script src=\"/assets/vendor/elk.bundled.js\" defer></script>\n", "")
	if err != nil {
		return nil, nil, err
	}
	bootstrap := "    <script>window.__ARCH_VIEW_EXPORT__ = " + string(bundleData) + ";</script>\n    <script>\n" + app + "\n    </script>"
	template, err = replaceRequired(template, `    <script src="/assets/app.js?v=20260826-manual-routing" defer></script>`, bootstrap)
	if err != nil {
		return nil, nil, err
	}
	template = strings.ReplaceAll(template, "__ARCH_VIEW_MODEL_ID__", html.EscapeString(value.ModelID))
	template = strings.ReplaceAll(template, "__ARCH_VIEW_SOURCE_ENABLED__", "false")
	template = strings.ReplaceAll(template, "__ARCH_VIEW_REANALYSIS_ENABLED__", "false")
	template = strings.ReplaceAll(template, "LOCAL ARCHITECTURE SESSION", "SELF-CONTAINED ARCHITECTURE EXPORT")
	return []byte(template), provenance, nil
}

func normalizeNewlines(value string) string {
	return strings.ReplaceAll(value, "\r\n", "\n")
}

func replaceRequired(value, old, replacement string) (string, error) {
	if !strings.Contains(value, old) {
		return "", analysis.NewHostError(analysis.ErrHostFailure, "HTML export template is incompatible with the viewer shell", map[string]any{"missing_fragment": old})
	}
	return strings.Replace(value, old, replacement, 1), nil
}

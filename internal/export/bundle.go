package export

import (
	"fmt"
	"path"
	"strings"

	"github.com/buffo/arch-view/internal/viewer"
	"github.com/evanw/esbuild/pkg/api"
)

const embeddedViewerNamespace = "arch-view-embedded"

// bundleViewer compiles the native viewer module graph from the same
// embedded assets used by the live server. The virtual filesystem keeps
// exports independent of the checkout and makes unresolved imports fail at
// build time instead of becoming external browser dependencies.
func bundleViewer() ([]byte, error) {
	result := api.Build(api.BuildOptions{
		EntryPoints: []string{"app.js"},
		Bundle:      true,
		Format:      api.FormatIIFE,
		Platform:    api.PlatformBrowser,
		Target:      api.ES2020,
		Write:       false,
		LogLevel:    api.LogLevelSilent,
		Plugins: []api.Plugin{{
			Name:  "embedded-viewer-assets",
			Setup: setupEmbeddedViewerPlugin,
		}},
	})
	if len(result.Errors) > 0 {
		message := result.Errors[0].Text
		if location := result.Errors[0].Location; location != nil {
			message = fmt.Sprintf("%s:%d:%d: %s", location.File, location.Line, location.Column, message)
		}
		return nil, fmt.Errorf("embedded viewer bundle failed: %s", message)
	}
	if len(result.OutputFiles) != 1 {
		return nil, fmt.Errorf("embedded viewer bundle produced %d outputs", len(result.OutputFiles))
	}
	return result.OutputFiles[0].Contents, nil
}

func setupEmbeddedViewerPlugin(build api.PluginBuild) {
	build.OnResolve(api.OnResolveOptions{Filter: ".*"}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
		if args.Importer == "" {
			if args.Path == "app.js" {
				return api.OnResolveResult{Path: "app.js", Namespace: embeddedViewerNamespace}, nil
			}
			return rejectEmbeddedImport(args.Path)
		}
		if !strings.HasPrefix(args.Path, ".") {
			return rejectEmbeddedImport(args.Path)
		}
		name := path.Clean(path.Join(path.Dir(args.Importer), args.Path))
		if !embeddedViewerAsset(name) {
			return rejectEmbeddedImport(args.Path)
		}
		return api.OnResolveResult{Path: name, Namespace: embeddedViewerNamespace}, nil
	})
	build.OnLoad(api.OnLoadOptions{Filter: ".*", Namespace: embeddedViewerNamespace}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
		if !embeddedViewerAsset(args.Path) {
			return api.OnLoadResult{Errors: []api.Message{{Text: "embedded viewer asset is not allowlisted: " + args.Path}}}, nil
		}
		data, err := viewer.Asset(args.Path)
		if err != nil {
			return api.OnLoadResult{Errors: []api.Message{{Text: err.Error()}}}, nil
		}
		contents := string(data)
		return api.OnLoadResult{Contents: &contents, Loader: api.LoaderJS}, nil
	})
}

func embeddedViewerAsset(name string) bool {
	if name == "app.js" || name == "layout_request.js" || name == "graph_route.js" {
		return true
	}
	return strings.HasPrefix(name, "app/") && path.Clean(name) == name && strings.HasSuffix(name, ".js")
}

func rejectEmbeddedImport(name string) (api.OnResolveResult, error) {
	return api.OnResolveResult{Errors: []api.Message{{Text: "external viewer import is not allowed: " + name}}}, nil
}

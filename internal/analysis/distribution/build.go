package distribution

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// BuildEntrypoint builds one Go command with explicit target and toolchain
// inputs. It never invokes a shell.
func BuildEntrypoint(ctx context.Context, request BuildRequest) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(request.RepositoryRoot) == "" {
		return fmt.Errorf("build repository root is required")
	}
	if strings.TrimSpace(request.Entrypoint) == "" {
		return fmt.Errorf("build entrypoint is required")
	}
	if strings.TrimSpace(request.OutputPath) == "" {
		return fmt.Errorf("build output path is required")
	}
	if strings.TrimSpace(request.GoCommand) == "" {
		return fmt.Errorf("build Go command is required")
	}
	if strings.TrimSpace(request.GOOS) == "" || strings.TrimSpace(request.GOARCH) == "" {
		return fmt.Errorf("build target requires GOOS and GOARCH")
	}
	cgoEnabled := request.CGOEnabled
	if cgoEnabled == "" {
		cgoEnabled = "1"
	}
	if cgoEnabled == "1" && (strings.TrimSpace(request.CCCommand) == "" || strings.TrimSpace(request.CXXCommand) == "") {
		return fmt.Errorf("cgo builds require explicit C and C++ compiler commands")
	}

	if err := os.MkdirAll(filepath.Dir(request.OutputPath), 0o755); err != nil {
		return fmt.Errorf("create build output directory: %w", err)
	}

	command := exec.CommandContext(ctx, request.GoCommand,
		"build",
		"-trimpath",
		"-buildvcs=false",
		"-ldflags=-buildid=",
		"-o", request.OutputPath,
		request.Entrypoint,
	)
	command.Dir = request.RepositoryRoot
	command.Env = buildEnvironment(os.Environ(), map[string]string{
		"GOOS":        request.GOOS,
		"GOARCH":      request.GOARCH,
		"CGO_ENABLED": cgoEnabled,
		"CC":          request.CCCommand,
		"CXX":         request.CXXCommand,
		"GOFLAGS":     "",
	})

	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Run(); err != nil {
		message := strings.TrimSpace(output.String())
		if message == "" {
			return fmt.Errorf("build %s for %s failed: %w", request.Entrypoint, request.GOOS+"-"+request.GOARCH, err)
		}
		return fmt.Errorf("build %s for %s failed: %w: %s", request.Entrypoint, request.GOOS+"-"+request.GOARCH, err, message)
	}
	return nil
}

func buildEnvironment(base []string, values map[string]string) []string {
	result := make([]string, 0, len(base)+len(values))
	for _, entry := range base {
		keep := true
		for key := range values {
			if strings.HasPrefix(entry, key+"=") {
				keep = false
				break
			}
		}
		if keep {
			result = append(result, entry)
		}
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		result = append(result, key+"="+values[key])
	}
	return result
}

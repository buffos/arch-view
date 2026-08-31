package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	appversion "github.com/buffo/arch-view/internal/version"
)

func TestVersionCommandReportsApplicationIdentity(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if got, want := strings.TrimSpace(stdout.String()), "arch-view 0.1.0 (commit unknown, built unknown, build local)"; got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("version wrote stderr: %s", stderr.String())
	}
}

func TestVersionFlagReportsApplicationIdentityWithoutInitializingACommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.HasPrefix(strings.TrimSpace(stdout.String()), "arch-view 0.1.0") {
		t.Fatalf("version flag output = %q", stdout.String())
	}
}

func TestVersionCommandSupportsMachineReadableOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"version", "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var got appversion.Info
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode version output: %v; output=%s", err, stdout.String())
	}
	if got.Application != "arch-view" || got.Version != "0.1.0" || got.BuildID != "local" {
		t.Fatalf("version info = %#v", got)
	}
}

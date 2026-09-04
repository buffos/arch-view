//go:build windows

package application

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestWindowsReplacementFailurePreservesConfigurationAndSession(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeApplicationFile(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: concept\n---\n")
	service := New(root)
	saved, err := service.SaveProfileAs(ctx, domain.Profile{Name: "Original"}, "", "working", "", "create", nil)
	if err != nil {
		t.Fatal(err)
	}
	before, err := service.SelectProfile(ctx, "", "project:working")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".archview.json")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	widePath, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	// Allow reads/writes, but omit FILE_SHARE_DELETE so MoveFileEx cannot replace
	// the document. This is an OS failure, not a mocked store or writer.
	handle, err := syscall.CreateFile(widePath, syscall.GENERIC_READ, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if handle != syscall.InvalidHandle {
			_ = syscall.CloseHandle(handle)
		}
	}()
	updated := saved.Profiles[0]
	updated.Name = "Recovered"
	_, err = service.SaveProfile(ctx, updated, saved.Revision, "replace", nil)
	denied := errors.Is(err, syscall.Errno(5)) || errors.Is(err, syscall.Errno(32))
	if !hasApplicationCode(err, "okf_configuration_write_failed") || !denied {
		t.Fatalf("expected denied replacement, got %v", err)
	}
	preserved, err := os.ReadFile(path)
	if err != nil || string(preserved) != string(original) {
		t.Fatalf("failed replacement changed original file: %v", err)
	}
	current, err := service.Session(ctx, "")
	if err != nil || !reflect.DeepEqual(before, current) {
		t.Fatalf("failed replacement changed session: %v", err)
	}
	leftovers, err := filepath.Glob(filepath.Join(root, ".archview.json.tmp-*"))
	if err != nil || len(leftovers) != 0 {
		t.Fatalf("temporary files remain: %v %v", leftovers, err)
	}
	if err := syscall.CloseHandle(handle); err != nil {
		t.Fatal(err)
	}
	handle = syscall.InvalidHandle
	retried, err := service.SaveProfile(ctx, updated, saved.Revision, "replace", nil)
	if err != nil || len(retried.Profiles) != 1 || retried.Profiles[0].Name != "Recovered" {
		t.Fatalf("replacement did not recover after handle closed: %+v %v", retried, err)
	}
}

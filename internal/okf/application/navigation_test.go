package application

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCancelledNavigationDoesNotMutateReadySession(t *testing.T) {
	root := t.TempDir()
	writeApplicationFile(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: concept\n---\nContent")
	service := New(root)
	if _, err := service.Session(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	service.mu.RLock()
	before := service.sessionLockedRead("")
	service.mu.RUnlock()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.SetNavigation(ctx, "", 5, false); err == nil {
		t.Fatal("cancelled request succeeded")
	}
	if _, err := service.Focus(ctx, "", "root"); err == nil {
		t.Fatal("cancelled focus succeeded")
	}
	service.mu.RLock()
	after := service.sessionLockedRead("")
	service.mu.RUnlock()
	if !reflect.DeepEqual(before, after) {
		t.Fatal("cancelled request changed ready session")
	}
}

func TestDefaultSessionAliasWorksForDetail(t *testing.T) {
	root := t.TempDir()
	writeApplicationFile(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: concept\n---\nContent")
	service := New(root)
	if _, err := service.Session(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	detail, err := service.Detail(context.Background(), "", "root")
	if err != nil || detail.ConceptID != "root" {
		t.Fatalf("default session alias lost: %#v %v", detail, err)
	}
}

func TestBackWithoutHistoryRejectsWithoutPublishingProjection(t *testing.T) {
	root := t.TempDir()
	writeApplicationFile(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: concept\n---\nContent")
	service := New(root)
	view, err := service.Back(context.Background(), "")
	if !hasApplicationCode(err, "okf_navigation_history_empty") || view.Projection != nil {
		t.Fatalf("empty Back must reject: %#v %v", view, err)
	}
}

func TestTopLevelPreservesSettingsAndBackReturnsToBranch(t *testing.T) {
	root := t.TempDir()
	writeApplicationFile(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: topic\n---\n")
	service := New(root)
	ctx := context.Background()
	if _, err := service.Session(ctx, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetNavigation(ctx, "", 3, true); err != nil {
		t.Fatal(err)
	}
	branch, err := service.Focus(ctx, "", "root")
	if err != nil {
		t.Fatal(err)
	}
	top, err := service.TopLevel(ctx, "")
	if err != nil || top.Navigation.FocusRoot != "" || top.Navigation.Depth != 3 || !top.Navigation.Full || !top.Navigation.CanGoBack || top.ProfileID != branch.ProfileID || top.BundleID != branch.BundleID {
		t.Fatalf("top-level changed context: %+v %v", top, err)
	}
	if _, err := service.TopLevel(ctx, ""); err != nil {
		t.Fatal(err)
	}
	back, err := service.Back(ctx, "")
	if err != nil || back.Navigation.FocusRoot != "root" || back.Navigation.Depth != 3 || !back.Navigation.Full {
		t.Fatalf("Back did not restore branch: %+v %v", back, err)
	}
	if _, err := service.Back(ctx, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Back(ctx, ""); !hasApplicationCode(err, "okf_navigation_history_empty") {
		t.Fatalf("history not exhausted: %v", err)
	}
}

func TestBackAvailabilityReflectsHistoryNotAncestry(t *testing.T) {
	root := t.TempDir()
	writeApplicationFile(t, filepath.Join(root, ".okf", "root.md"), "---\ntype: concept\n---\nContent")
	service := New(root)
	ctx := context.Background()
	if _, err := service.Session(ctx, ""); err != nil {
		t.Fatal(err)
	}
	focused, err := service.Focus(ctx, "", "root")
	if err != nil || !focused.Navigation.CanGoBack || !focused.Projection.Navigation.CanGoBack {
		t.Fatalf("missing history flag: %#v %v", focused, err)
	}
	back, err := service.Back(ctx, "")
	if err != nil || back.Navigation.CanGoBack || back.Projection.Navigation.CanGoBack {
		t.Fatalf("consumed history still enabled: %#v %v", back, err)
	}
	// A restored focused session can have ancestry without previous navigation.
	service.mu.Lock()
	session := service.sessionLocked("")
	session.FocusRoot, session.LastSnapshot = "root", nil
	service.mu.Unlock()
	restored, err := service.Session(ctx, "")
	if err != nil || len(restored.Navigation.Breadcrumbs) == 0 || restored.Navigation.CanGoBack || restored.Projection.Navigation.CanGoBack {
		t.Fatalf("ancestry mistaken for history: %#v %v", restored, err)
	}
}

package service

import (
	"path/filepath"
	"testing"

	"unifiedsubscriptionproxy/internal/platform/store"
)

func TestResolveDispatchUsesAllowedPackageRoute(t *testing.T) {
	dir := t.TempDir()
	svc := New(store.NewFileStore(filepath.Join(dir, "platform.json")))

	result, err := svc.ResolveDispatch("gpt-reasoning", "usp_demo_key")
	if err != nil {
		t.Fatalf("ResolveDispatch returned error: %v", err)
	}
	if result.Provider == "" {
		t.Fatalf("expected provider to be selected")
	}
	if result.UpstreamModel == "" {
		t.Fatalf("expected upstream model to be selected")
	}
}

package main

import (
	"os"
	"strings"
	"testing"
)

func TestGoAndRustShareMinimumRouteContract(t *testing.T) {
	raw, err := os.ReadFile("../proxy-core-rust/src/proxy/server.rs")
	if err != nil {
		t.Fatalf("failed to read rust proxy server: %v", err)
	}
	content := string(raw)
	for _, route := range []string{`"/healthz"`, `"/v1/chat/completions"`} {
		if !strings.Contains(content, route) {
			t.Fatalf("expected rust proxy core to expose route %s", route)
		}
	}
}

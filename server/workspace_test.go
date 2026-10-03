package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceRejectsPathsOutsideIt(t *testing.T) {
	tmpl := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpl, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := newWorkspace(tmpl, filepath.Join(t.TempDir(), "ws"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.Reset(); err != nil {
		t.Fatal(err)
	}
	if got, err := ws.ReadFile("a.go"); err != nil || got != "package a\n" {
		t.Fatalf("ReadFile(a.go) = %q, %v", got, err)
	}
	for _, p := range []string{"../secret", "/etc/passwd", "x/../../y"} {
		if _, err := ws.ReadFile(p); err == nil {
			t.Errorf("ReadFile(%q) should be rejected", p)
		}
		if err := ws.Apply(map[string]string{p: "x"}); err == nil {
			t.Errorf("Apply(%q) should be rejected", p)
		}
	}
}

func TestWorkspaceResetRestoresTemplate(t *testing.T) {
	tmpl := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpl, "a.go"), []byte("v1"), 0o644)
	ws, _ := newWorkspace(tmpl, filepath.Join(t.TempDir(), "ws"))
	_ = ws.Reset()
	if err := ws.Apply(map[string]string{"a.go": "v2", "b.go": "new"}); err != nil {
		t.Fatal(err)
	}
	_ = ws.Reset()
	files, _ := ws.Files()
	if len(files) != 1 || files["a.go"] != "v1" {
		t.Fatalf("after reset files = %v", files)
	}
}
